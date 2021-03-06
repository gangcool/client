// Auto-generated by avdl-compiler v1.3.22 (https://github.com/keybase/node-avdl-compiler)
//   Input file: avdl/keybase1/merkle.avdl

package keybase1

import (
	"github.com/keybase/go-framed-msgpack-rpc/rpc"
	context "golang.org/x/net/context"
)

type MerkleRootAndTime struct {
	Root       MerkleRootV2 `codec:"root" json:"root"`
	UpdateTime Time         `codec:"updateTime" json:"updateTime"`
	FetchTime  Time         `codec:"fetchTime" json:"fetchTime"`
}

func (o MerkleRootAndTime) DeepCopy() MerkleRootAndTime {
	return MerkleRootAndTime{
		Root:       o.Root.DeepCopy(),
		UpdateTime: o.UpdateTime.DeepCopy(),
		FetchTime:  o.FetchTime.DeepCopy(),
	}
}

type GetCurrentMerkleRootArg struct {
	FreshnessMsec int `codec:"freshnessMsec" json:"freshnessMsec"`
}

type MerkleInterface interface {
	// * getCurrentMerkleRoot gets the current-most Merkle root from the keybase server.
	// * The caller can specify how stale a result can be with freshnessMsec.
	// * If 0 is specified, then any amount of staleness is OK. If -1 is specified, then
	// * we force a GET and a round-trip.
	GetCurrentMerkleRoot(context.Context, int) (MerkleRootAndTime, error)
}

func MerkleProtocol(i MerkleInterface) rpc.Protocol {
	return rpc.Protocol{
		Name: "keybase.1.merkle",
		Methods: map[string]rpc.ServeHandlerDescription{
			"getCurrentMerkleRoot": {
				MakeArg: func() interface{} {
					ret := make([]GetCurrentMerkleRootArg, 1)
					return &ret
				},
				Handler: func(ctx context.Context, args interface{}) (ret interface{}, err error) {
					typedArgs, ok := args.(*[]GetCurrentMerkleRootArg)
					if !ok {
						err = rpc.NewTypeError((*[]GetCurrentMerkleRootArg)(nil), args)
						return
					}
					ret, err = i.GetCurrentMerkleRoot(ctx, (*typedArgs)[0].FreshnessMsec)
					return
				},
				MethodType: rpc.MethodCall,
			},
		},
	}
}

type MerkleClient struct {
	Cli rpc.GenericClient
}

// * getCurrentMerkleRoot gets the current-most Merkle root from the keybase server.
// * The caller can specify how stale a result can be with freshnessMsec.
// * If 0 is specified, then any amount of staleness is OK. If -1 is specified, then
// * we force a GET and a round-trip.
func (c MerkleClient) GetCurrentMerkleRoot(ctx context.Context, freshnessMsec int) (res MerkleRootAndTime, err error) {
	__arg := GetCurrentMerkleRootArg{FreshnessMsec: freshnessMsec}
	err = c.Cli.Call(ctx, "keybase.1.merkle.getCurrentMerkleRoot", []interface{}{__arg}, &res)
	return
}
