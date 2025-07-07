package collectors

type Collector interface {
	Name() string
	Status() string
}

type CollectorRegistry struct {
	collectors []Collector
}

func NewRegistry() *CollectorRegistry {
	return &CollectorRegistry{
		collectors: make([]Collector, 0),
	}
}

func (r *CollectorRegistry) Register(collector Collector) {
	r.collectors = append(r.collectors, collector)
}

func (r *CollectorRegistry) GetCollectors() []Collector {
	return r.collectors
}

type MockCollector struct {
	name   string
	status string
}

func NewMockCollector(name, status string) Collector {
	return &MockCollector{
		name:   name,
		status: status,
	}
}

func (c *MockCollector) Name() string {
	return c.name
}

func (c *MockCollector) Status() string {
	return c.status
}