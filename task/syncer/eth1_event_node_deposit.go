package task_syncer

import (
	"context"
	"math/big"

	"github.com/SergeevDmitry/eth2-balance-service/dao"
	"github.com/SergeevDmitry/eth2-balance-service/pkg/utils"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"gorm.io/gorm"
)

func (task *Task) fetchNodeDepositEvents(start, end uint64) error {
	iterDeposited, err := task.nodeDepositContract.FilterDeposited(&bind.FilterOpts{
		Start:   start,
		End:     &end,
		Context: context.Background(),
	})
	if err != nil {
		return err
	}
	for iterDeposited.Next() {
		txHashStr := iterDeposited.Event.Raw.TxHash.String()
		pubkeyStr := hexutil.Encode(iterDeposited.Event.ValidatorPubkey)

		validator, err := dao.GetValidator(task.db, pubkeyStr)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		// already synced
		if err == nil {
			continue
		}

		validator.NodeAddress = iterDeposited.Event.Node.String()
		validator.NodeDepositAmount = new(big.Int).Div(iterDeposited.Event.Amount, big.NewInt(1e9)).Uint64()
		// only support common node in v2
		validator.NodeType = utils.NodeTypeCommon

		validator.Status = utils.ValidatorStatusDeposited
		validator.Pubkey = pubkeyStr
		validator.DepositTxHash = txHashStr
		validator.DepositSignature = hexutil.Encode(iterDeposited.Event.ValidatorSignature)
		validator.PoolAddress = iterDeposited.Event.Pool.String()

		err = dao.UpOrInValidator(task.db, validator)
		if err != nil {
			return err
		}
	}

	iterStaked, err := task.nodeDepositContract.FilterStaked(&bind.FilterOpts{
		Start:   start,
		End:     &end,
		Context: context.Background(),
	})
	if err != nil {
		return err
	}
	for iterStaked.Next() {
		txHashStr := iterStaked.Event.Raw.TxHash.String()
		pubkeyStr := hexutil.Encode(iterStaked.Event.ValidatorPubkey)

		validator, err := dao.GetValidator(task.db, pubkeyStr)
		if err != nil {
			return err
		}

		if len(validator.StakeTxHash) != 0 {
			continue
		}

		validator.Status = utils.ValidatorStatusStaked
		validator.StakeTxHash = txHashStr
		validator.StakeBlockHeight = iterStaked.Event.Raw.BlockNumber

		err = dao.UpOrInValidator(task.db, validator)
		if err != nil {
			return err
		}
	}

	return nil
}
