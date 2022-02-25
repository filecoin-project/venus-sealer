package docgen

import (
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	datatransfer "github.com/filecoin-project/go-data-transfer"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/exitcode"
	network2 "github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	stype "github.com/filecoin-project/venus-sealer/types"
	types "github.com/filecoin-project/venus/venus-shared/types/messager"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-graphsync"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
	"unicode"
)

var ExampleValues = map[reflect.Type]interface{}{
	reflect.TypeOf(auth.Permission("")): auth.Permission("write"),
	reflect.TypeOf(""):                  "string value",
	reflect.TypeOf(uint64(42)):          uint64(42),
	reflect.TypeOf(uint(3)):             uint(3),
	reflect.TypeOf(byte(7)):             byte(7),
	reflect.TypeOf([]byte{}):            []byte("byte array"),
}

func addExample(v interface{}) {
	ExampleValues[reflect.TypeOf(v)] = v
}

func init() {
	const fixedUUID = "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
	uid := uuid.MustParse(fixedUUID)
	c, err := cid.Decode("bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4")
	if err != nil {
		panic(err)
	}
	ExampleValues[reflect.TypeOf(c)] = c
	addr, err := address.NewIDAddress(1234)
	if err != nil {
		panic(err)
	}

	ExampleValues[reflect.TypeOf(addr)] = addr

	pid, err := peer.Decode("12D3KooWGzxzKZYveHXtpG6AsrUJBcWxHBFS2HsEoGTxrMLvKXtf")
	if err != nil {
		panic(err)
	}
	addExample(uid)
	addExample(&uid)
	addExample(pid)
	addExample(&pid)
	addExample(network2.Version14)
	clientEvent := retrievalmarket.ClientEventDealAccepted
	addExample(bitfield.NewFromSet([]uint64{5}))
	addExample(abi.RegisteredSealProof_StackedDrg32GiBV1_1)
	addExample(abi.RegisteredPoStProof_StackedDrgWindow32GiBV1)
	addExample(abi.ChainEpoch(10101))
	addExample(crypto.SigTypeBLS)
	addExample(storiface.UnpaddedByteIndex(1023477))
	addExample(stores.ID(ExampleValue("init", reflect.TypeOf(uuid.UUID{}), nil).(uuid.UUID).String()))
	addExample(int64(9))
	addExample(12.3)
	addExample(123)
	addExample(uintptr(0))
	addExample(abi.MethodNum(1))
	addExample(exitcode.ExitCode(0))
	addExample(stype.Empty)
	addExample(api.SectorState(stype.PreCommit1))
	addExample(storiface.ErrUnknown)
	addExample(crypto.DomainSeparationTag_ElectionProofProduction)
	addExample(true)
	addExample(abi.UnpaddedPieceSize(1024))
	addExample(abi.UnpaddedPieceSize(1024).Padded())
	addExample(abi.DealID(5432))
	addExample(abi.SectorNumber(9))
	addExample(types.UnKnown)
	addExample(abi.SectorSize(32 * 1024 * 1024 * 1024))
	addExample(network.Connected)
	addExample(time.Minute)
	addExample(graphsync.RequestID(4))
	addExample(datatransfer.TransferID(3))
	addExample(datatransfer.Ongoing)
	addExample(clientEvent)
	addExample(&clientEvent)
	addExample(retrievalmarket.ClientEventDealAccepted)
	addExample(retrievalmarket.DealStatusNew)
	addExample(network.ReachabilityPublic)
	addExample(map[string]int{"name": 42})
	addExample(map[string]time.Time{"name": time.Unix(1615243938, 0).UTC()})
	addExample(map[string]*pubsub.TopicScoreSnapshot{
		"/blocks": {
			TimeInMesh:               time.Minute,
			FirstMessageDeliveries:   122,
			MeshMessageDeliveries:    1234,
			InvalidMessageDeliveries: 3,
		},
	})
	addExample(map[string]metrics.Stats{
		"12D3KooWSXmXLJmBR1M7i9RW9GQPNUhZSzXKzxDHWtAgNuJAbyEJ": {
			RateIn:   100,
			RateOut:  50,
			TotalIn:  174000,
			TotalOut: 12500,
		},
	})
	addExample(map[protocol.ID]metrics.Stats{
		"/fil/hello/1.0.0": {
			RateIn:   100,
			RateOut:  50,
			TotalIn:  174000,
			TotalOut: 12500,
		},
	})
	maddr, err := multiaddr.NewMultiaddr("/ip4/52.36.61.156/tcp/1347/p2p/12D3KooWFETiESTf1v4PGUvtnxMAcEFMzLZbJGg4tjWfGEimYior")
	if err != nil {
		panic(err)
	}
	// because reflect.TypeOf(maddr) returns the concrete type...
	ExampleValues[reflect.TypeOf(struct{ A multiaddr.Multiaddr }{}).Field(0).Type] = maddr
	si := uint64(12)
	addExample(&si)
	addExample(retrievalmarket.DealID(5))
	addExample(abi.ActorID(1000))
	addExample(map[abi.SectorNumber]string{
		123: "can't acquire read lock",
	})
	addExample([]abi.SectorNumber{123, 124})
	addExample(map[string]interface{}{"abc": 123})

	addExample(storiface.FTUnsealed)
	addExample(storiface.PathSealing)
	addExample(stype.TTAddPiece)
	addExample(map[string][]stype.SealedRef{"10": {ExampleValue("init", reflect.TypeOf(stype.SealedRef{}), nil).(stype.SealedRef)}})
	addExample(map[api.SectorState]int{
		ExampleValue("init", reflect.TypeOf(api.SectorState("")), nil).(api.SectorState): 0})
	addExample(map[stores.ID][]stores.Decl{
		ExampleValue("init", reflect.TypeOf(stores.ID("")), nil).(stores.ID): {ExampleValue("init", reflect.TypeOf(stores.Decl{}), nil).(stores.Decl)}})
	addExample(map[stores.ID]string{ExampleValue("init", reflect.TypeOf(stores.ID("")), nil).(stores.ID): "local path"})
	addExample(map[stores.ID]string{ExampleValue("init", reflect.TypeOf(stores.ID("")), nil).(stores.ID): "local path"})
	addExample(constants.MinerAPIVersion0)
	addExample(map[uuid.UUID][]storiface.WorkerJob{
		ExampleValue("init", reflect.TypeOf(uuid.UUID{}), nil).(uuid.UUID): {ExampleValue("init", reflect.TypeOf(storiface.WorkerJob{}), nil).(storiface.WorkerJob)}})
	addExample(map[stype.TaskType]map[abi.RegisteredSealProof]storiface.Resources{
		ExampleValue("init", reflect.TypeOf(stype.TTAddPiece), nil).(stype.TaskType): {
			ExampleValue("init", reflect.TypeOf(abi.RegisteredSealProof_StackedDrg2KiBV1), nil).(abi.RegisteredSealProof): ExampleValue("init", reflect.TypeOf(storiface.Resources{}), nil).(storiface.Resources),
		}})
	addExample(map[uuid.UUID]storiface.WorkerStats{
		ExampleValue("init", reflect.TypeOf(uuid.UUID{}), nil).(uuid.UUID): ExampleValue("init", reflect.TypeOf(storiface.WorkerStats{}), nil).(storiface.WorkerStats)})
}

func GetAPIType(name, pkg string) (i interface{}, t reflect.Type, permStruct []reflect.Type) {
	i = &api.StorageMinerStruct{}
	t = reflect.TypeOf(new(struct{ api.StorageMiner })).Elem()
	permStruct = append(permStruct, reflect.TypeOf(api.StorageMinerStruct{}.Internal))
	permStruct = append(permStruct, reflect.TypeOf(api.CommonStruct{}.Internal))
	return
}

func ExampleValue(method string, t, parent reflect.Type) interface{} {
	v, ok := ExampleValues[t]
	if ok {
		return v
	}
	switch t.Kind() {
	case reflect.Slice:
		out := reflect.New(t).Elem()
		v := ExampleValue(method, t.Elem(), t)
		out = reflect.Append(out, reflect.ValueOf(v))
		return out.Interface()
	case reflect.Chan:
		return ExampleValue(method, t.Elem(), nil)
	case reflect.Struct:
		es := exampleStruct(method, t, parent)
		v := reflect.ValueOf(es).Elem().Interface()
		ExampleValues[t] = v
		return v
	case reflect.Array:
		out := reflect.New(t).Elem()
		for i := 0; i < t.Len(); i++ {
			out.Index(i).Set(reflect.ValueOf(ExampleValue(method, t.Elem(), t)))
		}
		return out.Interface()
	case reflect.Ptr:
		if t.Elem().Kind() == reflect.Struct {
			es := exampleStruct(method, t.Elem(), t)
			// ExampleValues[t] = es
			return es
		}
	case reflect.Interface:
		if t.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			return fmt.Errorf("empty error")
		}
		return struct{}{}
	}

	_, _ = fmt.Fprintf(os.Stderr, "Warnning: No example value for type: %s (method '%s')\n", t, method)
	return reflect.New(t).Interface()
}

func exampleStruct(method string, t, parent reflect.Type) interface{} {
	ns := reflect.New(t)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Type == parent {
			continue
		}
		if strings.Title(f.Name) == f.Name {
			ns.Elem().Field(i).Set(reflect.ValueOf(ExampleValue(method, f.Type, t)))
		}
	}

	return ns.Interface()
}

type Visitor struct {
	Root    string
	Methods map[string]ast.Node
}

func (v *Visitor) Visit(node ast.Node) ast.Visitor {
	st, ok := node.(*ast.TypeSpec)
	if !ok {
		return v
	}

	if st.Name.Name != v.Root {
		return nil
	}

	iface := st.Type.(*ast.InterfaceType)
	for _, m := range iface.Methods.List {
		if len(m.Names) > 0 {
			v.Methods[m.Names[0].Name] = m
		}
	}

	return v
}

const NoComment = "There are not yet any comments for this method."

func ParseApiASTInfo(apiFile, iface, pkg, dir string) (comments map[string]string, groupDocs map[string]string) { //nolint:golint
	fset := token.NewFileSet()
	apiDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Println("./api filepath absolute error: ", err)
		return
	}
	apiFile, err = filepath.Abs(apiFile)
	if err != nil {
		fmt.Println("filepath absolute error: ", err, "file:", apiFile)
		return
	}
	pkgs, err := parser.ParseDir(fset, apiDir, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		fmt.Println("parse error: ", err)
		return
	}

	ap := pkgs[pkg]

	f := ap.Files[apiFile]

	cmap := ast.NewCommentMap(fset, f, f.Comments)

	v := &Visitor{iface, make(map[string]ast.Node)}
	ast.Walk(v, ap)

	comments = make(map[string]string)
	groupDocs = make(map[string]string)
	for mn, node := range v.Methods {
		filteredComments := cmap.Filter(node).Comments()
		if len(filteredComments) == 0 {
			comments[mn] = NoComment
		} else {
			for _, c := range filteredComments {
				if strings.HasPrefix(c.Text(), "MethodGroup:") {
					parts := strings.Split(c.Text(), "\n")
					groupName := strings.TrimSpace(parts[0][12:])
					comment := strings.Join(parts[1:], "\n")
					groupDocs[groupName] = comment

					break
				}
			}

			l := len(filteredComments) - 1
			if len(filteredComments) > 1 {
				l = len(filteredComments) - 2
			}
			last := filteredComments[l].Text()
			if !strings.HasPrefix(last, "MethodGroup:") {
				comments[mn] = last
			} else {
				comments[mn] = NoComment
			}
		}
	}
	return comments, groupDocs
}

type MethodGroup struct {
	GroupName string
	Header    string
	Methods   []*Method
}

type Method struct {
	Comment         string
	Name            string
	InputExample    string
	ResponseExample string
}

func MethodGroupFromName(mn string) string {
	i := strings.IndexFunc(mn[1:], func(r rune) bool {
		return unicode.IsUpper(r)
	})
	if i < 0 {
		return ""
	}
	return mn[:i+1]
}
