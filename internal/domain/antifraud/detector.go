package antifraud

type Detector struct {
	threshold        uint64
	normalWeight     uint64
	suspiciousWeight uint64
	current          map[string]uint64
}

func NewDetector(threshold, normalWeight, suspiciousWeight uint64) *Detector {
	if threshold == 0 {
		threshold = 1
	}
	if normalWeight == 0 {
		normalWeight = 10
	}
	if suspiciousWeight == 0 {
		suspiciousWeight = 1
	}

	return &Detector{
		threshold:        threshold,
		normalWeight:     normalWeight,
		suspiciousWeight: suspiciousWeight,
		current:          make(map[string]uint64),
	}
}
func (d *Detector) Weight(query string) uint64 {
	count := d.current[query] + 1
	d.current[query] = count

	if count > d.threshold {
		return d.suspiciousWeight
	}

	return d.normalWeight
}
func (d *Detector) Count(query string) uint64 {
	return d.current[query]
}
func (d *Detector) Reset() {
	clear(d.current)
}
