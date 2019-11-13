package softfork

import (
	"context"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/config"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/elastic_cache"
	"github.com/NavExplorer/navexplorer-indexer-go/internal/service/softfork/signal"
	"github.com/NavExplorer/navexplorer-indexer-go/pkg/explorer"
	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
)

type Rewinder struct {
	elastic       *elastic_cache.Index
	signalRepo    *signal.Repository
	blocksInCycle uint
}

func NewRewinder(elastic *elastic_cache.Index, signalRepo *signal.Repository, blocksInCycle uint) *Rewinder {
	return &Rewinder{elastic, signalRepo, blocksInCycle}
}

func (r *Rewinder) Rewind(height uint64) error {
	log.Infof("Rewinding soft fork index to height: %d", height)

	if err := r.elastic.DeleteHeightGT(height, elastic_cache.SignalIndex.Get()); err != nil {
		return err
	}

	for idx, s := range SoftForks {
		SoftForks[idx] = &explorer.SoftFork{
			MetaData:  s.MetaData,
			Name:      s.Name,
			SignalBit: s.SignalBit,
			State:     explorer.SoftForkDefined,
		}
	}

	start := uint64(1)
	end := uint64(r.blocksInCycle)

	for {
		if height == 0 || end >= height {
			break
		}

		signals := r.signalRepo.GetSignals(start, end)

		for _, s := range signals {
			for _, sf := range s.SoftForks {
				softFork := SoftForks.GetSoftFork(sf)
				if softFork.IsOpen() {
					softFork.SignalHeight = end
					softFork.State = explorer.SoftForkStarted
					cycleIndex := explorer.GetCycleForHeight(s.Height, r.blocksInCycle)

					var cycle *explorer.SoftForkCycle
					if cycle = softFork.GetCycle(cycleIndex); cycle == nil {
						softFork.Cycles = append(softFork.Cycles, explorer.SoftForkCycle{Cycle: cycleIndex, BlocksSignalling: 0})
						cycle = softFork.GetCycle(cycleIndex)
					}
					cycle.BlocksSignalling++
				}
			}
		}

		for _, s := range SoftForks {
			if s.State == explorer.SoftForkStarted && s.LatestCycle().BlocksSignalling >= explorer.GetQuorum(r.blocksInCycle) {
				s.State = explorer.SoftForkLockedIn
				s.LockedInHeight = end
				s.ActivationHeight = end + uint64(r.blocksInCycle)
			}
			if s.State == explorer.SoftForkLockedIn && height >= s.ActivationHeight {
				s.State = explorer.SoftForkActive
			}
		}

		start, end = func(start uint64, end uint64, height uint64) (uint64, uint64) {
			start += uint64(config.Get().SoftForkBlockCycle)
			end += uint64(config.Get().SoftForkBlockCycle)
			if end > height {
				end = height
			}
			return start, end
		}(start, end, height)
	}

	bulk := r.elastic.Client.Bulk()
	for _, sf := range SoftForks {
		bulk.Add(elastic.NewBulkUpdateRequest().
			Index(elastic_cache.SoftForkIndex.Get()).
			Id(sf.MetaData.Id).
			Doc(sf))
	}

	if bulk.NumberOfActions() > 0 {
		if _, err := bulk.Do(context.Background()); err != nil {
			log.WithError(err).Fatal("Failed to rewind soft forks")
		}
	}

	return nil
}
