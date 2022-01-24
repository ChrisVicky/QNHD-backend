package models

import (
	"fmt"
	"qnhd/pkg/logging"
	"qnhd/pkg/upload"
	"qnhd/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"gorm.io/gorm"
)

type Floor struct {
	Model
	Uid         uint64 `json:"uid"`
	PostId      uint64 `json:"post_id"`
	Content     string `json:"content"`
	Nickname    string `json:"nickname"`
	ImageURL    string `json:"image_url"`
	ReplyTo     uint64 `json:"reply_to" `
	ReplyToName string `json:"reply_to_name"`
	SubTo       uint64 `json:"sub_to" gorm:"default:0"`
	LikeCount   uint64 `json:"like_count"`
	DisCount    uint64 `json:"-"`
}

type LogFloorLike struct {
	Uid     uint64 `json:"uid"`
	FloorId uint64 `json:"floor_id"`
}

func (LogFloorLike) TableName() string {
	return "log_floor_like"
}

type LogFloorDis struct {
	Uid     uint64 `json:"uid"`
	FloorId uint64 `json:"floor_id"`
}

func (LogFloorDis) TableName() string {
	return "log_floor_dis"
}

type FloorResponse struct {
	Floor
	SubFloors   []FloorResponse `json:"sub_floors"`
	SubFloorCnt int             `json:"sub_floor_cnt"`
	IsLike      bool            `json:"is_like"`
	IsDis       bool            `json:"is_dis"`
	IsOwner     bool            `json:"is_owner"`
}

func (Floor) TableName() string {
	return "floors"
}

func (f *Floor) geneResponse(uid string, searchSubFloors bool) (FloorResponse, error) {
	var fr = FloorResponse{
		Floor:     *f,
		IsLike:    IsLikeFloorByUid(uid, util.AsStrU(f.Id)),
		IsDis:     IsDisFloorByUid(uid, util.AsStrU(f.Id)),
		IsOwner:   IsOwnFloorByUid(uid, util.AsStrU(f.Id)),
		SubFloors: []FloorResponse{},
	}
	if searchSubFloors {
		// 处理回复本条楼层的楼层
		rps, err := getFloorHighLikeShortReplyResponses(util.AsStrU(f.Id), uid)
		if err != nil {
			return fr, nil
		}
		fr.SubFloors = rps
		// 获取子楼层总数
		fr.SubFloorCnt = getFloorSubFloorCount(util.AsStrU(f.Id))
	}
	return fr, nil
}

// 将楼层数组转为返回结果数组
func transFloorsToResponses(floor *[]Floor, uid string, searchSubFloors bool) ([]FloorResponse, error) {
	var frs = []FloorResponse{}
	for _, f := range *floor {
		fr, err := f.geneResponse(uid, searchSubFloors)
		if err != nil {
			return frs, err
		}
		frs = append(frs, fr)
	}
	return frs, nil
}

const OWNER_NAME = "Owner"

// var FLOOR_NAME = []string{
// 	"Angus", "Bertram", "Conrad", "Devin", "Emmanuel", "Fitzgerald", "Gregary", "Herbert", "Ingram", "Joyce", "Kelly", "Leo", "Morton", "Nathaniel", "Orville", "Payne", "Quintion", "Regan", "Sean", "Tracy", "Uriah", "Valentine", "Walker", "Xavier", "Yves", "Zachary",
// }
const FLOOR_NAME = "测试"

// 根据id返回
func GetFloor(floorId string) (Floor, error) {
	var floor Floor
	err := db.Where("id = ?", floorId).First(&floor).Error
	return floor, err
}

// 返回单个楼层带Response
func GetFloorResponse(floorId, uid string) (FloorResponse, error) {
	var ret FloorResponse
	floor, err := GetFloor(floorId)
	if err != nil {
		return ret, err
	}
	return floor.geneResponse(uid, true)
}

// 缩略返回帖子内楼层，即返回5条
func GetShortFloorResponsesInPost(postId, uid string) ([]FloorResponse, error) {
	var floors []Floor
	if err := db.Where("post_id = ? AND reply_to = 0", postId).Order("created_at").Limit(5).Find(&floors).Error; err != nil {
		return nil, err
	}
	return transFloorsToResponses(&floors, uid, true)
}

// 分页返回帖子里的楼层
func GetFloorResponsesInPost(c *gin.Context, postId, uid string) ([]FloorResponse, error) {
	var floors []Floor
	if err := db.Where("post_id = ? AND reply_to = 0", postId).Order("created_at").Scopes(util.Paginate(c)).Find(&floors).Error; err != nil {
		return nil, err
	}
	return transFloorsToResponses(&floors, uid, true)
}

// 返回楼层内最高赞的5条楼层
func getFloorHighLikeShortReplyResponses(floorId, uid string) ([]FloorResponse, error) {
	var floors []Floor
	// 按照点赞降序，创建时间降序
	err := db.Where("sub_to = ?", floorId).Order("like_count DESC, created_at DESC").Limit(5).Find(&floors).Error
	if err != nil {
		return []FloorResponse{}, err
	}
	return transFloorsToResponses(&floors, uid, false)
}

func getFloorSubFloorCount(floorId string) int {
	var ret int64
	db.Model(&Floor{}).Where("sub_to = ?", floorId).Count(&ret)
	return int(ret)
}

func getCommentCount(postId uint64) int {
	var ret int64
	db.Model(&Floor{}).Where("post_id = ?", postId).Count(&ret)
	return int(ret)
}

// 分页返回楼层内的回复
func GetFloorReplyResponses(c *gin.Context, floorId, uid string) ([]FloorResponse, error) {
	var floors []Floor
	err := db.Where("sub_to = ?", floorId).Order("created_at").Scopes(util.Paginate(c)).Find(&floors).Error
	if err != nil {
		return []FloorResponse{}, err
	}
	return transFloorsToResponses(&floors, uid, false)
}

// 添加楼层评论
func AddFloor(maps map[string]interface{}) (uint64, error) {
	var post Post
	var nickname string
	uid := maps["uid"].(uint64)
	postId := maps["postId"].(uint64)
	// 先找到post主人
	if err := db.First(&post, postId).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}

	if post.Uid == uid {
		nickname = OWNER_NAME
	} else {
		// 还有可能已经发过言
		var floor Floor
		if err := db.Where("uid = ? AND post_id = ?", uid, postId).First(&floor).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, err
			}
		}
		if floor.Id > 0 {
			nickname = floor.Nickname
		} else {
			var cnt int64
			// 除去owner
			if err := db.Table("floors").Where("post_id = ? AND uid <> ?", postId, post.Uid).Distinct("uid").Count(&cnt).Error; err != nil {
				return 0, err
			}
			// nickname = FLOOR_NAME[cnt]
			nickname = fmt.Sprintf("%v%d", FLOOR_NAME, cnt)
		}
	}
	var newFloor = Floor{
		Uid:      uid,
		PostId:   postId,
		Content:  maps["content"].(string),
		Nickname: nickname,
		ImageURL: maps["image_url"].(string),
	}
	if err := db.Select("uid", "post_id", "content", "nickname", "image_url").Create(&newFloor).Error; err != nil {
		return 0, err
	}
	// 如果不是回复自己的帖子，通知帖子主人
	if post.Uid != uid {
		if err := addUnreadFloor(post.Uid, newFloor.Id); err != nil {
			return 0, err
		}
	}
	// 对帖子的tag增加记录, 当是树洞帖才会有
	if post.Type == POST_HOLE {
		if err := addTagLogInPost(post.Id, TAG_ADDFLOOR); err != nil {
			return 0, err
		}
	}

	return newFloor.Id, nil
}

// 添加楼层回复
func ReplyFloor(maps map[string]interface{}) (uint64, error) {
	var post Post
	var nickname string
	uid := maps["uid"].(uint64)
	// 判断存在floor
	floorId := maps["replyToFloor"].(uint64)
	var toFloor Floor
	if err := db.First(&toFloor, floorId).Error; err != nil {
		return 0, err
	}
	postId := toFloor.PostId
	// 先找到post主人
	if err := db.First(&post, postId).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}

	if post.Uid == uid {
		nickname = OWNER_NAME
	} else {
		// 还有可能已经发过言
		var floor Floor
		if err := db.Where("uid = ? AND post_id = ?", uid, postId).First(&floor).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, err
			}
		}
		if floor.Id > 0 {
			nickname = floor.Nickname
		} else {
			var cnt int64
			// 除去owner
			if err := db.Table("floors").Where("post_id = ? AND uid <> ?", postId, post.Uid).Distinct("uid").Count(&cnt).Error; err != nil {
				return 0, err
			}
			// nickname = FLOOR_NAME[cnt]
			nickname = fmt.Sprintf("%v%d", FLOOR_NAME, cnt)
		}
	}

	var newFloor = Floor{
		Uid:         uid,
		PostId:      toFloor.PostId,
		Content:     maps["content"].(string),
		Nickname:    nickname,
		ImageURL:    maps["image_url"].(string),
		ReplyTo:     toFloor.Id,
		ReplyToName: toFloor.Nickname,
	}
	// 判断子楼层
	// 如果没有subto，说明回复的不是子楼层
	if toFloor.SubTo == 0 {
		newFloor.SubTo = toFloor.Id
	} else {
		newFloor.SubTo = toFloor.SubTo
	}

	if err := db.Select("uid", "post_id", "content", "nickname", "image_url", "reply_to", "reply_to_name", "sub_to").Create(&newFloor).Error; err != nil {
		return 0, err
	}
	// 如果不是回复自己的楼层，通知楼层主人和回复的人
	var subToFloor, _ = GetFloor(util.AsStrU(newFloor.SubTo))
	if subToFloor.Uid != uid {
		addUnreadFloor(post.Uid, newFloor.Id)
	}
	if toFloor.Uid != uid {
		addUnreadFloor(post.Uid, toFloor.Id)
	}
	// 对帖子的tag增加记录, 当是树洞帖才会有
	if post.Type == POST_HOLE {
		if err := addTagLogInPost(post.Id, TAG_ADDFLOOR); err != nil {
			return 0, err
		}
	}
	return newFloor.Id, nil
}

func DeleteFloorByAdmin(uid, floorId string) (uint64, error) {
	var floor = Floor{}

	if err := db.Where("id = ?", floorId).First(&floor).Error; err != nil {
		return 0, err
	}
	// 首先判断是否有权限
	var post, _ = GetPost(util.AsStrU(floor.PostId))
	// 如果能删，要么是超管 要么是湖底帖且是湖底管理员
	// 如果不是超管
	if !RequireRight(uid, UserRight{Super: true}) {
		return 0, fmt.Errorf("无权删除")
	}
	// 湖底帖且是湖底管理员
	if !(post.Type == POST_HOLE && RequireRight(uid, UserRight{StuAdmin: true})) {
		return 0, fmt.Errorf("无权删除")
	}
	if err := deleteFloor(&floor); err != nil {
		return 0, err
	}
	return floor.Id, nil
}

func DeleteFloorByUser(uid, floorId string) (uint64, error) {
	var floor = Floor{}
	if err := db.Where("uid = ? AND id = ?", uid, floorId).First(&floor).Error; err != nil {
		return 0, err
	}
	if err := deleteFloor(&floor); err != nil {
		return 0, err
	}
	return floor.Id, nil
}

func deleteFloor(floor *Floor) error {
	var imgs []string
	if floor.ImageURL != "" {
		imgs = append(imgs, floor.ImageURL)
	}
	// 判断是否有子楼层
	var floors []Floor
	db.Where("sub_to").Find(&floors)
	for _, f := range floors {
		if f.ImageURL != "" {
			imgs = append(imgs, f.ImageURL)
		}
	}

	// 删除本地文件
	if err := upload.DeleteImageUrls(imgs); err != nil {
		logging.Error(err.Error())
		return err
	}

	if err := db.Where("id = ? OR sub_to = ?", floor.Id, floor.Id).Delete(&Floor{}).Error; err != nil {
		return err
	}
	return nil
}

func DeleteFloorsInPost(tx *gorm.DB, postId uint64) error {
	if tx == nil {
		tx = db
	}
	var floors []Floor
	var imgs = []string{}
	if err := tx.Where("post_id = ?", postId).Find(&floors).Error; err != nil {
		return err
	}
	for _, f := range floors {
		if f.ImageURL != "" {
			imgs = append(imgs, f.ImageURL)
		}
	}
	// 删除本地文件
	if err := upload.DeleteImageUrls(imgs); err != nil {
		logging.Error(err.Error())
		return err
	}
	if err := tx.Where("post_id = ?", postId).Delete(&Floor{}).Error; err != nil {
		return err
	}
	return nil
}

// 点赞楼层
func LikeFloor(floorId string, uid string) (uint64, error) {
	var log LogFloorLike
	// 首先判断点没点过赞
	if err := db.Where("uid = ? AND floor_id = ?", uid, floorId).Find(&log).Error; err != nil {
		return 0, err
	}
	if log.Uid > 0 {
		return 0, fmt.Errorf("已被点赞")
	}

	log.Uid = util.AsUint(uid)
	log.FloorId = util.AsUint(floorId)
	if err := db.Create(&log).Error; err != nil {
		return 0, err
	}
	// 更新楼的likes
	var floor Floor
	if err := db.Where("id = ?", floorId).First(&floor).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&floor).Update("like_count", floor.LikeCount+1).Error; err != nil {
		return 0, err
	}

	UndisFloor(floorId, uid)
	return floor.LikeCount, nil
}

// 取消点赞楼层
func UnlikeFloor(floorId string, uid string) (uint64, error) {
	var log LogFloorLike
	// 首先判断点没点过赞
	if err := db.Where("uid = ? AND floor_id = ?", uid, floorId).Find(&log).Error; err != nil {
		return 0, err
	}

	if log.Uid == 0 {
		return 0, fmt.Errorf("未被点赞")
	}

	if err := db.Where("uid = ? AND floor_id = ?", uid, floorId).Delete(&log).Error; err != nil {
		return 0, err
	}

	// 更新楼的likes
	var floor Floor
	if err := db.Where("id = ?", floorId).First(&floor).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&floor).Update("like_count", floor.LikeCount-1).Error; err != nil {
		return 0, err
	}

	return floor.LikeCount, nil
}

// 点踩楼层
func DisFloor(floorId string, uid string) (uint64, error) {
	var log LogFloorDis
	// 首先判断点没点过踩
	if err := db.Where("uid = ? AND floor_id = ?", uid, floorId).Find(&log).Error; err != nil {
		return 0, err
	}
	if log.Uid > 0 {
		return 0, fmt.Errorf("已被点踩")
	}

	log.Uid = util.AsUint(uid)
	log.FloorId = util.AsUint(floorId)
	if err := db.Create(&log).Error; err != nil {
		return 0, err
	}
	// 更新楼的likes
	var floor Floor
	if err := db.Where("id = ?", floorId).First(&floor).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&floor).Update("dis_count", floor.DisCount+1).Error; err != nil {
		return 0, err
	}
	UnlikeFloor(floorId, uid)
	return floor.DisCount, nil
}

func UndisFloor(floorId string, uid string) (uint64, error) {
	var log LogFloorDis
	// 首先判断点没点过踩
	if err := db.Where("uid = ? AND floor_id = ?", uid, floorId).Find(&log).Error; err != nil {
		return 0, err
	}
	if log.Uid == 0 {
		return 0, fmt.Errorf("未被点踩")
	}

	if err := db.Where("uid = ? AND floor_id = ?", uid, floorId).Delete(&log).Error; err != nil {
		return 0, err
	}

	// 更新楼的likes
	var floor Floor
	if err := db.Where("id = ?", floorId).First(&floor).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&floor).Update("dis_count", floor.DisCount-1).Error; err != nil {
		return 0, err
	}

	return floor.DisCount, nil
}

func IsLikeFloorByUid(uid, floorId string) bool {
	var log LogFloorLike
	if err := db.Where("uid = ? AND floor_id = ?", uid, floorId).Find(&log).Error; err != nil {
		logging.Error(err.Error())
		return false
	}
	return log.Uid > 0
}

func IsDisFloorByUid(uid, floorId string) bool {
	var log LogFloorDis
	if err := db.Where("uid = ? AND floor_id = ?", uid, floorId).Find(&log).Error; err != nil {
		logging.Error(err.Error())
		return false
	}
	return log.Uid > 0
}

func IsOwnFloorByUid(uid, floorId string) bool {
	var floor, err = GetFloor(floorId)
	if err != nil {
		return false
	}
	return util.AsStrU(floor.Uid) == uid
}
