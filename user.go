package services

import (
	"errors"
	"github.com/xinglou123/pkg/db/xorm"
	"github.com/xinglou123/pkg/os/conv"
	"github.com/xinglou123/pkg/os/md5"
	"github.com/xinglou123/pkg/utils"
	"github.com/xinglou123/pkg/validator"
	"sync"
)

type User struct {
	Id         int64  `xorm:"pk autoincr 'id'" form:"id" json:"id"`
	Username   string `xorm:"char(60)" form:"username" json:"username"`
	Nick       string `xorm:"char(60)" form:"nick" json:"nick"`
	Email      string `xorm:"char(50)" form:"email" json:"email"`
	Phone      string `xorm:"char(50)" form:"phone" json:"phone"`
	Sex        string `xorm:"char(20)" form:"sex"	json:"sex"`
	Password   string `xorm:"char(100)" form:"password" json:"-"`
	Salt       string `xorm:"char(100)"  json:"-"`
	Score      int64  `xorm:"int(20)"  form:"score" json:"score"`
	Status     string `xorm:"char(20)" form:"status" json:"status"`
	Avatar     string `xorm:"char(200)" form:"avatar" json:"avatar"`
	Created_at string `xorm:"DateTime created" form:"created_at" json:"created_at" time_format:"2006-01-02 15:04:05"`
	Updated_at string `xorm:"DateTime updated" form:"updated_at" json:"updated_at" time_format:"2006-01-02 15:04:05"`
}

// UserService.
var UserService = &userService{
	mutex: &sync.Mutex{},
}

//UserService ...
type userService struct {
	mutex *sync.Mutex
}

//登录服务,通过手机号/邮箱/用户名登录
func (service userService) Signin(kword string, passwd string) (user User, err error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	if len(kword) == 0 && len(passwd) == 0 {
		err = errors.New("请输入用户名、密码")
		return
	}

	if !validator.IsPassword(passwd) {
		err = errors.New("字母开头，允许6-20字节，允许字母数字特殊字符")
		return
	}
	torm := xorm.MustDB()
	t := torm.Where("id>0")
	if validator.IsEmail(kword) {
		t = t.Where("email =  ?", kword)
	} else if validator.IsCellphone(kword) {
		t = t.Where("phone =  ?", kword)
	} else {
		t = t.Where("username =  ?", kword)
	}
	t.Get(&user)
	if user.Id == 0 {
		err = errors.New("该用户不存在")
		return
	}
	password := md5.CryptPassword(passwd, user.Salt)
	if password != user.Password {
		err = errors.New("密码不正确,请重试")
		return
	}
	return
}

//根据userId 获取用户编号
func (service userService) One(userId int64) (User, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	var u User
	if userId == 0 {
		return u, errors.New("缺少参数")
	}
	orm := xorm.MustDB()
	_, err := orm.Id(userId).Get(&u)
	return u, err
}

//根据userId 获取用户编号
func (service userService) Query(param map[string]interface{}) ([]User, *utils.Page, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	orm := xorm.MustDB()
	t := orm.Where("id>0")

	if username, ok := param["username"]; ok {
		t = t.Where("username like ?", "%"+username.(string)+"%")
	}
	if phone, ok := param["phone"]; ok {
		t = t.Where("phone like ?", "%"+phone.(string)+"%")
	}
	var users []User
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
	count, err := t.Limit(limit, offset).Desc("id").FindAndCount(&users)
	if err != nil {
		return users, nil, err
	}
	return users, utils.NewPage(page, limit, conv.Int(count)), err

}

//注册服务,注册后自动登录
func (service userService) Add(user User) (p User, err error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	if len(user.Username) == 0 && len(user.Email) == 0 && len(user.Phone) == 0 {
		err = errors.New("请输入用户名、email、手机号")
		return
	}
	if len(user.Email) > 0 {
		if !validator.IsEmail(user.Email) {
			err = errors.New("email 格式错误")
			return
		}

	}
	if len(user.Phone) > 0 {
		if !validator.IsCellphone(user.Phone) {
			err = errors.New("手机号格式错误")
			return
		}
	}
	if !validator.IsPassword(user.Password) {
		err = errors.New("字母开头，允许6-20字节，允许字母数字特殊字符")
		return
	}
	var u User
	torm := xorm.MustDB()
	t := torm.Where("id>0")
	if len(user.Username) > 0 {
		t = t.Where("username =  ?", user.Username)
	} else if len(user.Email) > 0 {
		t = t.Where("email =  ?", user.Email)
	} else if len(user.Phone) > 0 {
		t = t.Where("phone =  ?", user.Phone)
	}
	t.Get(&u)

	if u.Id > 0 {
		p = u
		return p, errors.New("该用户已存在")
	}
	salt := md5.GenRandomString(6, true)
	user.Password = md5.CryptPassword(user.Password, salt)
	user.Salt = salt

	orm := xorm.MustDB()
	session := orm.NewSession()
	defer session.Close()
	serr := session.Begin()

	_, serr = session.InsertOne(user)
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
	_, err = orm.Where("username = ? or phone= ?", user.Username, user.Phone).Get(&p)

	return p, err
}

/**
* 删除用户
*
* @param  int
 */
func (service userService) Delete(uid int64) (int64, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if uid == 0 {
		return 0, errors.New("不能删除该用户")
	}
	var yb User
	orm := xorm.MustDB()
	iid, err := orm.Id(uid).Delete(yb)
	return iid, err
}

/**
* 更新用户
*
* @param  int
* @return int
 */
func (service userService) Update(user User) (iid int64, err error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	if user.Id == 0 {
		return 0, errors.New("用户不存在")
	}
	orm := xorm.MustDB()
	session := orm.NewSession()
	defer session.Close()
	err = session.Begin()

	iid, err = session.Where("id = ?", user.Id).Update(user)
	if err != nil {
		err = errors.New("更新用户失败")
		session.Rollback()
		return iid, err
	}
	err = session.Commit()
	if err != nil {
		err = errors.New("更新用户失败")
		return 0, err
	}
	return iid, err

}

//user Count
func (service userService) Count(title string) (int64, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()

	var logger User
	orm := xorm.MustDB()
	total, err := orm.Count(logger)
	return total, err
}
