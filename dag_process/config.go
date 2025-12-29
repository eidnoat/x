package dag_process

type Config struct {
	ParallelLimit int
}

var (
	Cfg = &Config{ParallelLimit: 5}
)
