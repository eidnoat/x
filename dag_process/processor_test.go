package dag_process

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- 辅助函数 ---

// resetGraph 重置全局图 G，用于测试隔离
func resetGraph() *Graph {
	g := NewGraph()
	G.Store(g)
	return g
}

// --- Graph 编译测试 ---

func TestGraph_Compile_Success(t *testing.T) {
	g := resetGraph()
	g.Register("A", nil, nil)
	g.Register("B", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) { return 1, nil })

	if err := g.Compile(); err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
}

func TestGraph_Compile_Cycle(t *testing.T) {
	g := resetGraph()
	// A -> B -> A 循环依赖
	g.Register("A", []AttrID{"B"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) { return 1, nil })
	g.Register("B", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) { return 1, nil })

	if err := g.Compile(); err == nil {
		t.Fatal("Expected cycle error, got nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

func TestGraph_Compile_MissingDep(t *testing.T) {
	g := resetGraph()
	// A 依赖不存在的 MISSING
	g.Register("A", []AttrID{"MISSING"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) { return 1, nil })

	if err := g.Compile(); err == nil {
		t.Fatal("Expected missing dependency error, got nil")
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

// --- Processor 核心逻辑测试 ---

// TestProcessor_BasicChain 测试最简单的线性依赖: A -> B -> C
func TestProcessor_BasicChain(t *testing.T) {
	g := resetGraph()

	// A: 纯输入节点
	g.Register("A", nil, nil)
	// B = A * 2
	g.Register("B", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		return data["A"].(int) * 2, nil
	})
	// C = B + 10
	g.Register("C", []AttrID{"B"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		return data["B"].(int) + 10, nil
	})
	g.Compile()

	// 初始输入: A=5
	input := map[AttrID]any{"A": 5}
	p := NewProcessor(context.Background(), nil, input)

	if err := p.Process(context.Background()); err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if p.data["B"] != 10 {
		t.Errorf("Expected B=10, got %v", p.data["B"])
	}
	if p.data["C"] != 20 {
		t.Errorf("Expected C=20, got %v", p.data["C"])
	}
}

// TestProcessor_Diamond 测试菱形依赖: A->B, A->C, B+C->D
// 验证 Fan-out 和 Fan-in 逻辑
func TestProcessor_Diamond(t *testing.T) {
	g := resetGraph()
	g.Register("A", nil, nil)
	g.Register("B", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		return data["A"].(int) + 1, nil
	})
	g.Register("C", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		return data["A"].(int) + 2, nil
	})
	g.Register("D", []AttrID{"B", "C"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		return data["B"].(int) + data["C"].(int), nil
	})
	g.Compile()

	// Input A=10 -> B=11, C=12 -> D=23
	p := NewProcessor(context.Background(), nil, map[AttrID]any{"A": 10})
	if err := p.Process(context.Background()); err != nil {
		t.Fatal(err)
	}

	if val, ok := p.data["D"].(int); !ok || val != 23 {
		t.Errorf("Expected D=23, got %v", p.data["D"])
	}
}

// TestProcessor_InputOverride 验证 Input Map 优先级
// 场景: B 依赖 A，且 B 有计算逻辑。但在 NewProcessor 中直接指定了 B 的值。
// 预期: B 的值应直接使用 Input 中的值，忽略计算逻辑。
func TestProcessor_InputOverride(t *testing.T) {
	g := resetGraph()
	g.Register("A", nil, nil)
	g.Register("B", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		return data["A"].(int) * 2, nil
	})
	g.Compile()

	// A=10. 正常 B=20. 强制指定 B=999.
	input := map[AttrID]any{"A": 10, "B": 999}
	p := NewProcessor(context.Background(), nil, input)

	if err := p.Process(context.Background()); err != nil {
		t.Fatal(err)
	}

	if p.data["B"] != 999 {
		t.Errorf("Expected B=999 (Overridden), got %v", p.data["B"])
	}
}

// TestProcessor_Pruning 验证剪枝优化
// 场景: A->B。A 的值更新后与旧值相同，B 不应执行计算。
func TestProcessor_Pruning(t *testing.T) {
	g := resetGraph()
	var executionCount int32

	g.Register("A", nil, nil)
	g.Register("B", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		atomic.AddInt32(&executionCount, 1)
		return data["A"], nil
	})
	g.Compile()

	// 初始状态: A=10, B=10
	data := map[AttrID]any{"A": 10, "B": 10}
	// 输入: A=10 (值未变)
	input := map[AttrID]any{"A": 10}

	p := NewProcessor(context.Background(), data, input)
	if err := p.Process(context.Background()); err != nil {
		t.Fatal(err)
	}

	if count := atomic.LoadInt32(&executionCount); count != 0 {
		t.Errorf("Expected pruning (count=0), but B executed %d times", count)
	}
}

// TestProcessor_StateRetention 验证状态保持与连续执行
// 场景: 运行 Process -> 通过 NewProcessor 继承数据并传入新 input -> 再次运行 Process
func TestProcessor_StateRetention(t *testing.T) {
	g := resetGraph()
	g.Register("A", nil, nil)
	g.Register("B", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		return data["A"].(int) + 1, nil
	})
	g.Compile()

	// 第一次运行: A=1 -> B=2
	p := NewProcessor(context.Background(), nil, map[AttrID]any{"A": 1})
	if err := p.Process(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p.data["B"] != 2 {
		t.Errorf("Run 1: Expected B=2, got %v", p.data["B"])
	}

	// 模拟外部事件更新: A=10
	// 由于 Input 方法已移除，我们需要创建一个新的 Processor，
	// 并传入上一次运行后的 data 作为初始状态 (State Retention)，以及新的 input。
	p = NewProcessor(context.Background(), p.data, map[AttrID]any{"A": 10})

	// 第二次运行: 应基于新的 input 状态触发重算
	// B = 10 + 1 = 11
	if err := p.Process(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p.data["B"] != 11 {
		t.Errorf("Run 2: Expected B=11, got %v", p.data["B"])
	}
}

// TestProcessor_ErrorHandling 验证错误传播
func TestProcessor_ErrorHandling(t *testing.T) {
	g := resetGraph()
	g.Register("A", nil, nil)
	g.Register("B", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		return nil, errors.New("calculation error")
	})
	g.Compile()

	p := NewProcessor(context.Background(), nil, map[AttrID]any{"A": 1})
	if err := p.Process(context.Background()); err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// TestProcessor_Disconnected 验证断开的图 (多连通分量)
// Chain 1: A->B, Chain 2: C->D
func TestProcessor_Disconnected(t *testing.T) {
	g := resetGraph()
	g.Register("A", nil, nil)
	g.Register("B", []AttrID{"A"}, func(ctx context.Context, d map[AttrID]any, id AttrID) (any, error) { return d["A"].(int) + 1, nil })
	g.Register("C", nil, nil)
	g.Register("D", []AttrID{"C"}, func(ctx context.Context, d map[AttrID]any, id AttrID) (any, error) { return d["C"].(int) + 1, nil })
	g.Compile()

	input := map[AttrID]any{"A": 10, "C": 20}
	p := NewProcessor(context.Background(), nil, input)
	if err := p.Process(context.Background()); err != nil {
		t.Fatal(err)
	}

	if p.data["B"] != 11 || p.data["D"] != 21 {
		t.Error("Disconnected components processing failed")
	}
}

// TestProcessor_EmptyGraph 验证空图边界情况
func TestProcessor_EmptyGraph(t *testing.T) {
	g := resetGraph()
	g.Compile() // 空图

	// 不应 Panic
	p := NewProcessor(context.Background(), nil, nil)
	if err := p.Process(context.Background()); err != nil {
		t.Errorf("Empty graph process error: %v", err)
	}
}

// TestProcessor_ContextCancellation 验证 Context 取消/超时
func TestProcessor_ContextCancellation(t *testing.T) {
	g := resetGraph()
	g.Register("A", nil, nil)
	// B 会阻塞直到 context 取消
	g.Register("B", []AttrID{"A"}, func(ctx context.Context, data map[AttrID]any, id AttrID) (any, error) {
		select {
		case <-time.After(2 * time.Second): // 故意长时间等待
			return 0, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
	g.Compile()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	p := NewProcessor(ctx, nil, map[AttrID]any{"A": 1})

	start := time.Now()
	err := p.Process(ctx)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("Expected cancellation error, got nil")
	}
	// 验证确实是因为超时退出的，而不是跑完了
	if duration > 1*time.Second {
		t.Error("Process did not respect context timeout")
	}
}

// TestProcessor_HeavyConcurrency 大数据量并发测试
// 场景: 2000 个节点，20 层深，每层 100 个节点。验证死锁和正确性。
func TestProcessor_HeavyConcurrency(t *testing.T) {
	g := resetGraph()

	nodeCount := 2000
	layerCount := 20
	nodesPerLayer := nodeCount / layerCount

	// 构建图
	for i := 0; i < nodeCount; i++ {
		id := AttrID(fmt.Sprintf("N%d", i))
		layer := i / nodesPerLayer

		if layer == 0 {
			// 种子节点
			g.Register(id, nil, func(c context.Context, d map[AttrID]any, _ AttrID) (any, error) {
				return 1, nil
			})
		} else {
			// 依赖上一层的随机节点
			// 为了确保图是连通且复杂的，我们让它依赖上一层对应的节点 (i - nodesPerLayer)
			prevNodeId := AttrID(fmt.Sprintf("N%d", i-nodesPerLayer))
			g.Register(id, []AttrID{prevNodeId}, func(c context.Context, d map[AttrID]any, myId AttrID) (any, error) {
				val := d[prevNodeId].(int)
				// 模拟一点点计算耗时，增加并发冲突概率
				if rand.Intn(100) < 5 {
					time.Sleep(time.Microsecond)
				}
				return val + 1, nil
			})
		}
	}

	if err := g.Compile(); err != nil {
		t.Fatalf("Heavy graph compile failed: %v", err)
	}

	input := make(map[AttrID]any)
	for i := 0; i < nodesPerLayer; i++ {
		id := AttrID(fmt.Sprintf("N%d", i))
		input[id] = 1
	}

	// 临时提高并发限制
	originalLimit := Cfg.ParallelLimit
	Cfg.ParallelLimit = 100
	defer func() { Cfg.ParallelLimit = originalLimit }()

	// 使用 WaitGroup 并发运行多次，模拟真实高频请求
	var wg sync.WaitGroup
	concurrency := 5 // 同时跑 5 个 Process 请求

	start := time.Now()
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// 每个请求独立的 Processor
			p := NewProcessor(context.Background(), nil, input)
			if err := p.Process(context.Background()); err != nil {
				t.Errorf("Heavy process failed: %v", err)
			}

			// 验证结果 (最后一层应为 layerCount)
			lastNodeID := AttrID(fmt.Sprintf("N%d", nodeCount-1))
			if val, ok := p.data[lastNodeID].(int); !ok || val != layerCount {
				t.Errorf("Wrong result. Expected %d, got %v", layerCount, val)
			}
		}()
	}

	wg.Wait()
	t.Logf("Processed %d requests * %d nodes in %v", concurrency, nodeCount, time.Since(start))
}
