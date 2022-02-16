package models

import (
	"errors"
	"qnhd/pkg/util"
	"qnhd/request/twtservice"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UnreadNoticeResponse struct {
	Notice
	IsRead bool `json:"is_read" gorm:"-"`
}

type LogUnreadNotice struct {
	Uid      uint64 `json:"uid"`
	NoticeId uint64 `json:"floor_id"`
}

// 通知所有用户
func addUnreadNoticeToAllUser(notice *Notice) error {
	var (
		users   []User
		numbers []string
	)
	// 查询所有用户id
	if err := db.Where("is_user = true AND active = true").Find(&users).Error; err != nil {
		return err
	}
	var logs = []LogUnreadNotice{}
	for _, u := range users {
		logs = append(logs, LogUnreadNotice{u.Uid, notice.Id})
		numbers = append(numbers, u.Number)
	}
	if err := db.Create(logs).Error; err != nil {
		return err
	}

	return twtservice.NotifyNotice(notice.Sender, notice.Title, numbers...)
}

// 获取未读的所有notice
func GetUnreadNotices(c *gin.Context, uid uint64) ([]UnreadNoticeResponse, error) {
	var (
		logs    []LogUnreadNotice
		notices []Notice
		ret     = []UnreadNoticeResponse{}
	)
	if err := db.Where("uid = ?", uid).Find(&logs).Error; err != nil {
		return ret, err
	}
	if err := db.Scopes(util.Paginate(c)).Order("created_at DESC").Find(&notices).Error; err != nil {
		return ret, err
	}
	for _, n := range notices {
		var r = UnreadNoticeResponse{Notice: n}
		for _, log := range logs {
			if log.NoticeId == n.Id {
				r.IsRead = true
			}
		}
		ret = append(ret, r)
	}
	return ret, nil
}

// 是否通知已读
func IsReadNotice(uid, noticeId uint64) bool {
	var log LogUnreadNotice
	err := db.Where("uid = ? AND notice_id = ?", uid, noticeId).First(log).Error
	if err != nil && errors.Is(gorm.ErrRecordNotFound, err) {
		return true
	}
	return false
}