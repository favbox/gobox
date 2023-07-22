package stats

import (
	"testing"
	"time"

	"github.com/favbox/gosky/wind/pkg/common/test/assert"
	"github.com/favbox/gosky/wind/pkg/common/tracer/stats"
	"github.com/favbox/gosky/wind/pkg/common/tracer/traceinfo"
)

func TestUtil(t *testing.T) {
	assert.Assert(t, CalcEventCostUs(nil, nil) == 0)

	ti := traceinfo.NewTraceInfo()

	// nil context
	Record(ti, stats.HTTPStart, nil)
	Record(ti, stats.HTTPFinish, nil)

	st := ti.Stats()
	assert.Assert(t, st != nil)

	s, e := st.GetEvent(stats.HTTPStart), st.GetEvent(stats.HTTPFinish)
	assert.Assert(t, s == nil)
	assert.Assert(t, e == nil)

	// stats disabled
	Record(ti, stats.HTTPStart, nil)
	time.Sleep(time.Millisecond)
	Record(ti, stats.HTTPFinish, nil)

	st = ti.Stats()
	assert.Assert(t, st != nil)

	s, e = st.GetEvent(stats.HTTPStart), st.GetEvent(stats.HTTPFinish)
	assert.Assert(t, s == nil)
	assert.Assert(t, e == nil)

	// stats enabled
	st = ti.Stats()
	st.(interface{ SetLevel(stats.Level) }).SetLevel(stats.LevelBase)

	Record(ti, stats.HTTPStart, nil)
	time.Sleep(time.Millisecond)
	Record(ti, stats.HTTPFinish, nil)

	s, e = st.GetEvent(stats.HTTPStart), st.GetEvent(stats.HTTPFinish)
	assert.Assert(t, s != nil, s)
	assert.Assert(t, e != nil, e)
	assert.Assert(t, CalcEventCostUs(s, e) > 0)
}
