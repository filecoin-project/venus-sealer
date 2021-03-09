package sqlite

import (
	"github.com/filecoin-project/venus-sealer/types"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"os"
	"testing"
)

func setupDealRef(suffix string, t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("./deal_"+suffix), &gorm.Config{
		//Logger: logger.Default.LogMode(logger.Info), // 日志配置
	})
	if err != nil {
		t.Error(err)
	}
	db.AutoMigrate(&dealRef{})
	return db
}

func cleanDealRef(suffix string, t *testing.T) {
	os.Remove("./deal_" + suffix)
}

func Test_dealRefRepo_Get(t *testing.T) {
	db := setupDealRef("get_dealref", t)
	defer cleanDealRef("get_dealref", t)
	dRepo := newDealRefRepo(db)
	id1 := uuid.New().String()
	dRepo.Create(dealRef{
		Id:        id1,
		DealId:    12,
		SectorId:  1,
		PadOffset: 12,
		UnPadSize: 12,
	})
	id2 := uuid.New().String()
	err := dRepo.Create(dealRef{
		Id:        id2,
		DealId:    12,
		SectorId:  2,
		PadOffset: 13,
		UnPadSize: 13,
	}).Error
	if err != nil {
		t.Error(err)
	}

	refs, err := dRepo.Get(12)
	if err != nil {
		t.Error(err)
	}

	if len(refs.Refs) != 2 {
		t.Errorf("expect deal ref number is %d, but got %d", 2, len(refs.Refs))
	}

	if 1 != refs.Refs[0].SectorID {
		t.Errorf("expect sector %d but got %d", 1, refs.Refs[0].SectorID)
	}

	if 2 != refs.Refs[1].SectorID {
		t.Errorf("expect sector %d but got %d", 2, refs.Refs[0].SectorID)
	}
}

func Test_dealRefRepo_UnHas(t *testing.T) {
	db := setupDealRef("un_has_dealref", t)
	defer cleanDealRef("un_has_dealref", t)
	dRepo := newDealRefRepo(db)
	id1 := uuid.New().String()
	dRepo.Create(dealRef{
		Id:        id1,
		DealId:    12,
		SectorId:  1,
		PadOffset: 12,
		UnPadSize: 12,
	})
	has, err := dRepo.Has(13)
	if err != nil {
		t.Error(err)
	}

	if has {
		t.Errorf("query db fail ,expect none but got a deal")
	}
}

func Test_dealRefRepo_Has(t *testing.T) {
	db := setupDealRef("has_dealref", t)
	defer cleanDealRef("has_dealref", t)
	dRepo := newDealRefRepo(db)
	id1 := uuid.New().String()
	dRepo.Create(dealRef{
		Id:        id1,
		DealId:    12,
		SectorId:  1,
		PadOffset: 12,
		UnPadSize: 12,
	})
	has, err := dRepo.Has(12)
	if err != nil {
		t.Error(err)
	}

	if !has {
		t.Errorf("query db fail ,expect get deal but none")
	}
}

func Test_dealRefRepo_List(t *testing.T) {
	db := setupDealRef("deal_list", t)
	defer cleanDealRef("deal_list", t)
	dRepo := newDealRefRepo(db)
	id1 := uuid.New().String()
	dRepo.Create(dealRef{
		Id:        id1,
		DealId:    12,
		SectorId:  1,
		PadOffset: 12,
		UnPadSize: 12,
	})
	id2 := uuid.New().String()
	dRepo.Create(dealRef{
		Id:        id2,
		DealId:    13,
		SectorId:  2,
		PadOffset: 13,
		UnPadSize: 13,
	})
	id3 := uuid.New().String()
	dRepo.Create(dealRef{
		Id:        id3,
		DealId:    13,
		SectorId:  3,
		PadOffset: 15,
		UnPadSize: 15,
	})

	refs, err := dRepo.List()
	if err != nil {
		t.Error(err)
	}

	if len(refs) != 2 {
		t.Errorf("list should has %d dealid but got %d", 2, len(refs))
	}

	if len(refs[13]) != 2 {
		t.Errorf("deal 13 expect has %d sector but got %d", 2, len(refs[13]))
	}

	if len(refs[12]) != 1 {
		t.Errorf("deal 13 expect has %d sector but got %d", 1, len(refs[12]))
	}
}

func Test_dealRefRepo_Save(t *testing.T) {
	db := setupDealRef("save_deal", t)
	defer cleanDealRef("save_deal", t)
	dRepo := newDealRefRepo(db)

	err := dRepo.Save(1, types.SealedRef{
		SectorID: 12,
		Offset:   111,
		Size:     222,
	}, nil)

	if err != nil {
		t.Error(err)
	}

	refs, err := dRepo.List()
	if err != nil {
		t.Error(err)
	}

	if refs[1][0].SectorID != 12 {
		t.Errorf("expect sector id %d, but got %d", 12, refs[1][0].SectorID)
	}

	if refs[1][0].Offset != 111 {
		t.Errorf("expect sector Offset %d, but got %d", 111, refs[1][0].Offset)
	}

	if refs[1][0].Size != 222 {
		t.Errorf("expect sector Size %d, but got %d", 222, refs[1][0].Size)
	}
}
