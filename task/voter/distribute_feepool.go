package task_voter

import (
	"context"
	"fmt"
	"time"

	"github.com/SergeevDmitry/eth2-balance-service/pkg/utils"
	"github.com/sirupsen/logrus"
)

func (task *Task) distributeFeePool() error {
	balance, err := task.connection.Eth1Client().BalanceAt(context.Background(), task.feePoolAddress, nil)
	if err != nil {
		return err
	}

	if balance.Cmp(minDistributeAmountDeci.BigInt()) < 0 {
		return nil
	}

	logrus.Info("Will DistributeFee : ", balance.String())
	err = task.connection.LockAndUpdateTxOpts()
	if err != nil {
		return err
	}
	defer task.connection.UnlockTxOpts()

	tx, err := task.distributorContract.DistributeFee(task.connection.TxOpts(), balance)
	if err != nil {
		return err
	}
	logrus.Info("send DistributeFee tx hash: ", tx.Hash().String())

	retry := 0
	for {
		if retry > utils.RetryLimit {
			utils.ShutdownRequestChannel <- struct{}{}
			return fmt.Errorf("distributorContract.DistributeFee tx reach retry limit")
		}
		_, pending, err := task.connection.Eth1Client().TransactionByHash(context.Background(), tx.Hash())
		if err == nil && !pending {
			break
		} else {
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"err":  err.Error(),
					"hash": tx.Hash(),
				}).Warn("tx status")
			} else {
				logrus.WithFields(logrus.Fields{
					"hash":   tx.Hash(),
					"status": "pending",
				}).Warn("tx status")
			}
			time.Sleep(utils.RetryInterval)
			retry++
			continue
		}
	}
	logrus.WithFields(logrus.Fields{
		"tx": tx.Hash(),
	}).Info("DistributeFee tx send ok")

	return nil
}
