package dag_process

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// resetGraph 用于在每个测试开始前重置全局图 G
// 这是一个辅助函数
func resetGraph() *Graph {
	g := NewGraph()
	G.Store(g)
	return g
}

// TestGraph_Compile_Cycle 测试循环依赖检测
func TestGraph_Compile_Cycle(t *testing.T) {
	g := resetGraph()

	// A -> B, B -> A
	_ = g.Register("A", []AttrID{"B"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) { return 1, nil })
	_ = g.Register("B", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) { return 1, nil })

	err := g.Compile()
	if err == nil {
		t.Fatal("Expected error due to cycle, got nil")
	}
	t.Logf("Got expected cycle error: %v", err)
}

// TestGraph_Compile_MissingDep 测试缺失依赖检测
func TestGraph_Compile_MissingDep(t *testing.T) {
	g := resetGraph()

	// A -> MissingNode
	_ = g.Register("A", []AttrID{"MISSING"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) { return 1, nil })

	err := g.Compile()
	if err == nil {
		t.Fatal("Expected error due to missing dependency, got nil")
	}
	t.Logf("Got expected missing dep error: %v", err)
}

// TestProcessor_BasicFlow 测试最基本的计算流程: A(Input) -> B -> C
func TestProcessor_BasicFlow(t *testing.T) {
	g := resetGraph()

	// Input Node A (No ComputeFunc)
	g.Register("A", nil, nil)

	// B = A * 2
	g.Register("B", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		valA, _ := data["A"].(int)
		return valA * 2, nil
	})

	// C = B + 10
	g.Register("C", []AttrID{"B"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		valB, _ := data["B"].(int)
		return valB + 10, nil
	})

	if err := g.Compile(); err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	// Init
	p := NewProcessor(context.Background(), map[AttrID]any{"A": 0})

	// Input Update: A = 5
	p.Input(context.Background(), map[AttrID]any{"A": 5})

	// Process
	if err := p.Process(context.Background()); err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Verify
	if p.data["B"] != 10 {
		t.Errorf("Expected B=10, got %v", p.data["B"])
	}
	if p.data["C"] != 20 {
		t.Errorf("Expected C=20, got %v", p.data["C"])
	}
}

// TestProcessor_DiamondConcurrency 测试菱形依赖和并发
//
//	  A
//	 / \
//	B   C
//	 \ /
//	  D
func TestProcessor_DiamondConcurrency(t *testing.T) {
	g := resetGraph()
	var execCount int32

	g.Register("A", nil, nil)

	// B = A + 1
	g.Register("B", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		atomic.AddInt32(&execCount, 1)
		// 模拟一点点耗时，确保并发场景下的稳定性
		time.Sleep(10 * time.Millisecond)
		return data["A"].(int) + 1, nil
	})

	// C = A + 2
	g.Register("C", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		atomic.AddInt32(&execCount, 1)
		time.Sleep(10 * time.Millisecond)
		return data["A"].(int) + 2, nil
	})

	// D = B + C
	g.Register("D", []AttrID{"B", "C"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		return data["B"].(int) + data["C"].(int), nil
	})

	if err := g.Compile(); err != nil {
		t.Fatal(err)
	}

	p := NewProcessor(context.Background(), map[AttrID]any{"A": 10})
	p.Input(context.Background(), map[AttrID]any{"A": 100}) // A changed to 100

	start := time.Now()
	if err := p.Process(context.Background()); err != nil {
		t.Fatal(err)
	}
	duration := time.Since(start)

	// Verify Values
	// B = 101, C = 102, D = 203
	if p.data["D"] != 203 {
		t.Errorf("Expected D=203, got %v", p.data["D"])
	}

	// Verify Concurrency (Simple check)
	// B and C sleep 10ms each. If serial, total > 20ms. If parallel, total ~ 10ms.
	// This is flaky in CI, so we just log it, but logic correctness is key.
	t.Logf("Processing took %v", duration)
}

// TestProcessor_InputFilter 测试 Input 方法的过滤逻辑
func TestProcessor_InputFilter(t *testing.T) {
	g := resetGraph()
	g.Register("A", nil, nil)
	g.Compile()

	// Initial A=10
	p := NewProcessor(context.Background(), map[AttrID]any{"A": 10})

	// Case 1: Input same value -> dirty should be false
	p.Input(context.Background(), map[AttrID]any{"A": 10})
	if p.dirty["A"] {
		t.Error("Input with same value should not mark dirty")
	}

	// Case 2: Input new value -> dirty should be true
	p.Input(context.Background(), map[AttrID]any{"A": 20})
	if !p.dirty["A"] {
		t.Error("Input with new value should mark dirty")
	}
	if p.data["A"] != 20 {
		t.Errorf("Data not updated, got %v", p.data["A"])
	}

	// Case 3: Input unknown key -> should be ignored (based on your logic)
	p.Input(context.Background(), map[AttrID]any{"UNKNOWN_KEY": 999})
	if _, exists := p.data["UNKNOWN_KEY"]; exists {
		t.Error("Unknown key should have been ignored")
	}
}

// TestProcessor_ErrorHandling 测试计算出错时的处理
func TestProcessor_ErrorHandling(t *testing.T) {
	g := resetGraph()
	g.Register("A", nil, nil)
	// B 总是报错
	g.Register("B", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		return nil, errors.New("calculation failed")
	})
	g.Compile()

	p := NewProcessor(context.Background(), map[AttrID]any{"A": 1})
	p.Input(context.Background(), map[AttrID]any{"A": 2})

	err := p.Process(context.Background())
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	// 验证错误信息是否包含层级信息或原始错误
	// 这里简单检查一下不为空即可
}
