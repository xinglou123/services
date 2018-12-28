package services

import (
	"errors"
	"github.com/xinglou123/pkg/db/xorm"
	"github.com/xinglou123/pkg/os/conv"
	"github.com/xinglou123/pkg/utils"
	"sync"
)

type Logger struct {
	Id         int64   `xorm:"pk autoincr 'id'" form:"id" json:"id"`
	User_id    string  `xorm:"char(100)" form:"user_id" json:"user_id"`
	Path       string  `xorm:"char(100)" form:"path" json:"path"`
	Params     string  `xorm:"char(200)" form:"params" json:"params"`
	Method     string  `xorm:"char(100)" form:"method" json:"method"`
	Ip         string  `xorm:"char(100)" form:"ip" json:"ip"`
	Latency    float64 `xorm:"float(11)"  form:"latency" json:"latency"`
	Agent      string  `xorm:"char(200)" form:"agent" json:"agent"`
	Status     int     `xorm:"int" form:"status" json:"status"`
	Rtime      string  `xorm:"char(100)" form:"rtime" json:"rtime"`
	Created_at string  `xorm:"DateTime created" form:"created_at" json:"created_at" time_format:"2006-01-02 15:04:05"`
	Updated_at string  `xorm:"DateTime updated" form:"updated_at" json:"updated_at" time_format:"2006-01-02 15:04:05"`
}

// LoggerService
var LoggerService = &loggerService{
	mutex: &sync.Mutex{},
}

//TagTimeModel ...
type loggerService struct {
	mutex *sync.Mutex
}

//根据Id 获取
func (service *loggerService) One(uid int64) (*Logger, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if uid == 0 {
		return nil, errors.New("缺少参数")
	}
	var u Logger
	orm := xorm.MustDB()
	_, err := orm.Id(uid).Get(&u)
	return &u, err
}
func (service *loggerService) Query(param map[string]interface{}) ([]Logger, *utils.Page, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	orm := xorm.MustDB()
	t := orm.Where("id>0")

	if path, ok := param["path"]; ok {
		//存在
		if len(path.(string)) > 0 {
			t = t.Where("path like ?", "%"+path.(string)+"%")
		}
	}
	var loggers []Logger
	var page int = 1
	if pagek, ok := param["page"]; ok {
		page = conv.Int(pagek)
	}
	var limit int = 10
	if limitk, ok := param["limit"]; ok {
		limit = conv.Int(limitk)
		if limit == 0 {
			limit = 10
		}
	}
	offset := (page - 1) * limit
	count, err := t.Limit(limit, offset).Desc("id").FindAndCount(&loggers)
	if err != nil {
		return loggers, nil, err
	}
	return loggers, utils.NewPage(page, limit, conv.Int(count)), err
}

//注册服务,注册后自动登录
func (service *loggerService) Add(logger Logger) (p Logger, err error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	orm := xorm.MustDB()
	//orm.Where("path = ?", logger.Path).Get(&p)
	//
	//if p.Id > 0 {
	//	err = errors.New("该记录已存在")
	//	return p, err
	//}
	session := orm.NewSession()
	defer session.Close()
	serr := session.Begin()

	_, serr = session.InsertOne(logger)
	if serr != nil {
		err = errors.New("添加失败")
		session.Rollback()
		return p, err
	}
	serr = session.Commit()
	if serr != nil {
		err = errors.New("添加失败")
		return p, err
	}
	_, err = orm.Where("path = ?", logger.Path).Get(&p)

	return p, err

}

/**
* 删除
*
* @param  int
 */
func (service *loggerService) Delete(uid int64) (int64, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if uid == 0 {
		return 0, errors.New("不能删除该记录")
	}
	var yb Logger
	orm := xorm.MustDB()
	iid, err := orm.Id(uid).Delete(yb)
	return iid, err
}

/**
* 更新
*
* @param  int
* @return int
 */
func (service *loggerService) Update(logger Logger) (iid int64, err error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	if logger.Id == 0 {
		return 0, errors.New("不能修改该记录")
	}
	orm := xorm.MustDB()
	session := orm.NewSession()
	defer session.Close()
	err = session.Begin()

	iid, err = session.Where("id = ?", logger.Id).Update(logger)
	if err != nil {
		err = errors.New("更新失败")
		session.Rollback()
		return iid, err
	}
	err = session.Commit()
	if err != nil {
		err = errors.New("更新失败")
		return 0, err
	}
	return iid, err

}

//Count
func (service *loggerService) Count() (int64, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	var logger Logger
	orm := xorm.MustDB()
	total, err := orm.Count(logger)
	return total, err
}
