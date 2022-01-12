package models

import (
	"errors"
	"fmt"
	"qnhd/pkg/logging"
	"qnhd/pkg/upload"
	"qnhd/pkg/util"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PostCampusType int

const (
	CAMPUS_NONE PostCampusType = iota
	CAMPUS_OLD
	CAMPUS_NEW
)

type PostType int

const (
	POST_SCHOOL PostType = iota
	POST_HOLE
	POST_ALL
)

type Post struct {
	Model
	Uid uint64 `json:"-" gorm:"column:uid"`

	// 帖子分类
	Type         PostType       `json:"type"`
	DepartmentId uint64         `json:"-" gorm:"column:department_id;default:0"`
	Campus       PostCampusType `json:"campus"`
	Solved       int            `json:"solved" gorm:"defalut:0"`

	// 帖子内容
	Title   string `json:"title"`
	Content string `json:"content"`

	// 各种数量
	FavCount  uint64 `json:"fav_count" gorm:"defalut:0"`
	LikeCount uint64 `json:"like_count" gorm:"defalut:0"`
	DisCount  uint64 `json:"-" gorm:"defalut:0"`

	// 评分
	Rating uint64 `json:"rating" gorm:"default:0"`

	UpdatedAt string `json:"-" gorm:"default:null;"`
}

type LogPostFav struct {
	Model
	Uid    uint64 `json:"uid"`
	PostId uint64 `json:"post_id"`
}
type LogPostLike struct {
	Model
	Uid    uint64 `json:"uid"`
	PostId uint64 `json:"post_id"`
}
type LogPostDis struct {
	Model
	Uid    uint64 `json:"uid"`
	PostId uint64 `json:"post_id"`
}

type PostResponse struct {
	Post
	Tag          Tag             `json:"tag"`
	Floors       []FloorResponse `json:"floors"`
	CommentCount int             `json:"comment_count"`
	IsLike       bool            `json:"is_like"`
	IsDis        bool            `json:"is_dis"`
	IsFav        bool            `json:"is_fav"`
	IsOwner      bool            `json:"is_owner"`
	ImageUrls    []string        `json:"image_urls"`
	Department   Department      `json:"department"`
}

func (p *Post) geneResponse(uid string) (PostResponse, error) {
	var pr PostResponse
	tag, err := GetTagInPost(util.AsStrU(p.Id))
	if err != nil {
		return pr, err
	}
	frs, err := GetShortFloorResponsesInPost(util.AsStrU(p.Id), uid)
	if err != nil {
		return pr, err
	}
	imgs, err := GetImageInPost(util.AsStrU(p.Id))
	if err != nil {
		return pr, err
	}
	var depart Department
	if p.DepartmentId > 0 {
		d, err := GetDepartment(p.DepartmentId)
		if err != nil {
			return pr, err
		}
		depart = d
	}
	return PostResponse{
		Post:         *p,
		Tag:          tag,
		Floors:       frs,
		CommentCount: len(frs),
		IsLike:       IsLikePostByUid(uid, util.AsStrU(p.Id)),
		IsDis:        IsDisPostByUid(uid, util.AsStrU(p.Id)),
		IsFav:        IsFavPostByUid(uid, util.AsStrU(p.Id)),
		IsOwner:      IsOwnPostByUid(uid, util.AsStrU(p.Id)),
		ImageUrls:    imgs,
		Department:   depart,
	}, nil
}

func transPostsToResponses(posts *[]Post, uid string) ([]PostResponse, error) {
	var prs = []PostResponse{}
	for _, p := range *posts {
		pr, err := p.geneResponse(uid)
		if err != nil {
			return prs, err
		}
		prs = append(prs, pr)
	}
	return prs, nil
}

func GetPost(postId string) (Post, error) {
	var post Post
	err := db.Where("id = ?", postId).First(&post).Error
	return post, err
}

func GetPostResponseAndVisit(postId string, uid string) (PostResponse, error) {
	var post Post
	var pr PostResponse
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		return pr, err
	}
	if _, err := addVisitHistory(uid, postId); err != nil {
		return pr, err
	}
	if err := addTagLogInPost(util.AsUint(postId), TAG_VISIT); err != nil {
		return pr, err
	}
	return post.geneResponse(uid)
}

func GetPostResponses(c *gin.Context, uid string, maps map[string]interface{}) ([]PostResponse, error) {
	var posts []Post
	content := maps["content"].(string)
	postType := maps["type"].(PostType)
	departmentId, departOk := maps["department_id"].(string)
	solved, solvedOk := maps["solved"].(string)

	var d = db.Scopes(util.Paginate(c)).Where("CONCAT(title,content) LIKE ?", "%"+content+"%").Order("created_at DESC")
	// 校区 不为全部时加上区分
	if postType != POST_ALL {
		d = d.Where("type = ?", postType)
	}
	// 如果有部门要加上
	if departOk {
		d = d.Where("department_id = ?", departmentId)
	}
	// 如果要加上是否解决的字段
	if solvedOk {
		d = d.Where("solved = ?", solved)
	}

	if err := d.Find(&posts).Error; err != nil {
		return nil, err
	}
	return transPostsToResponses(&posts, uid)
}

func GetUserPostResponses(c *gin.Context, uid string) ([]PostResponse, error) {
	var posts []Post
	if err := db.Where("uid = ?", uid).Scopes(util.Paginate(c)).Find(&posts).Error; err != nil {
		return nil, err
	}
	return transPostsToResponses(&posts, uid)
}

func GetFavPostResponses(c *gin.Context, uid string) ([]PostResponse, error) {
	var posts []Post
	if err := db.Joins("JOIN log_post_fav ON posts.id = log_post_fav.post_id AND log_post_fav.deleted_at IS NULL").Scopes(util.Paginate(c)).Find(&posts).Error; err != nil {
		return nil, err
	}
	return transPostsToResponses(&posts, uid)
}

func GetHistoryPostResponses(c *gin.Context, uid string) ([]PostResponse, error) {
	var posts []Post
	var ids []string
	if err := db.Model(&LogVisitHistory{}).Where("uid = ?", uid).Distinct("post_id").Scopes(util.Paginate(c)).Scan(&ids).Error; err != nil {
		return nil, err
	}

	if err := db.Where("id IN (?)", ids).Scopes(util.Paginate(c)).Find(&posts).Error; err != nil {
		return nil, err
	}
	return transPostsToResponses(&posts, uid)
}

func AddPost(maps map[string]interface{}) (uint64, error) {
	var err error
	var post = &Post{
		Type:    maps["type"].(PostType),
		Uid:     maps["uid"].(uint64),
		Campus:  maps["campus"].(PostCampusType),
		Title:   maps["title"].(string),
		Content: maps["content"].(string),
	}
	if post.Type == POST_SCHOOL {
		imgs, img_ok := maps["image_urls"].([]string)
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Select("type", "uid", "campus", "title", "content").Create(post).Error; err != nil {
				return err
			}
			if img_ok {
				if err := AddImageInPost(tx, post.Id, imgs); err != nil {
					return err
				}
			}
			// 如果有tag_id
			tagId, ok := maps["tag_id"].(string)
			if ok {
				if err := AddPostWithTag(post.Id, tagId); err != nil {
					return err
				}
				// 对帖子的tag增加记录
				if err := addTagLog(util.AsUint(tagId), TAG_ADDPOST); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			upload.DeleteImageUrls(imgs)
			return 0, err
		}
	} else if post.Type == POST_HOLE {
		// 先对department_id进行查找，不存在要报错
		departId := maps["department_id"].(uint64)
		if err = db.Where("id = ?", departId).First(&Department{}).Error; err != nil {
			return 0, err
		}
		post.DepartmentId = departId
		imgs, img_ok := maps["image_urls"].([]string)
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Select("type", "uid", "campus", "title", "content", "department_id").Create(post).Error; err != nil {
				return err
			}

			if img_ok {
				if err := AddImageInPost(tx, post.Id, imgs); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			upload.DeleteImageUrls(imgs)
			return 0, err
		}
	} else {
		return 0, fmt.Errorf("invalid post type")
	}

	return post.Id, nil
}

func EditPostSolved(postId string, rating string) error {
	err := db.Model(&Post{}).Where("id = ?", postId).Updates(map[string]interface{}{
		"solved": 1,
		"rating": rating,
	}).Error
	return err
}

func DeletePostsUser(id, uid string) (uint64, error) {
	var post = Post{}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND uid = ?", id, uid).First(&post).Error; err != nil {
			return err
		}
		if err := tx.Delete(&post).Error; err != nil {
			return err
		}
		if err := DeleteTagInPost(tx, id); err != nil {
			return err
		}
		if err := DeleteFloorsInPost(tx, id); err != nil {
			return err
		}
		if err := DeleteImageInPost(tx, id); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return post.Id, nil
}

func DeletePostsAdmin(id string) (uint64, error) {
	var post = Post{}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).First(&post).Error; err != nil {
			return err
		}
		if err := tx.Delete(post).Error; err != nil {
			return err
		}
		if err := DeleteTagInPost(tx, id); err != nil {
			return err
		}
		if err := DeleteFloorsInPost(tx, id); err != nil {
			return err
		}
		if err := DeleteImageInPost(tx, id); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	return post.Id, nil
}

func FavPost(postId string, uid string) (uint64, error) {
	var exist = false
	var log = LogPostFav{Uid: util.AsUint(uid), PostId: util.AsUint(postId)}

	// 首先判断点没点过赞
	if err := db.Where(log).Find(&log).Error; err != nil {
		return 0, err
	}
	if log.Id > 0 {
		return 0, fmt.Errorf("已收藏")
	}

	if err := db.Unscoped().Where(log).Find(&log).Error; err != nil {
		return 0, err
	}

	exist = log.Id > 0
	if exist {
		if err := db.Unscoped().Model(&log).Update("deleted_at", gorm.Expr("NULL")).Error; err != nil {
			return 0, err
		}
	} else {
		if err := db.Select("uid", "post_id").Create(&log).Error; err != nil {
			return 0, err
		}
	}
	// 更新楼的likes
	var post Post
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&post).Update("fav_count", post.FavCount+1).Error; err != nil {
		return 0, err
	}
	return post.FavCount + 1, nil
}

func UnfavPost(postId string, uid string) (uint64, error) {
	uidint := util.AsUint(uid)
	postIdint := util.AsUint(postId)

	var exist = false
	var log = LogPostFav{Uid: uidint, PostId: postIdint}
	// 首先判断点没点过赞
	if err := db.Where(log).Find(&log).Error; err != nil {
		return 0, err
	}
	if log.Id == 0 {
		return 0, fmt.Errorf("未收藏")
	}

	if err := db.Where(log).Find(&log).Error; err != nil {
		return 0, err
	}
	exist = log.Id > 0
	if exist {
		if err := db.Delete(&log).Error; err != nil {
			return 0, err
		}
	}

	// 更新楼的likes
	var post Post
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&post).Update("fav_count", post.FavCount-1).Error; err != nil {
		return 0, err
	}
	return post.FavCount - 1, nil
}

func LikePost(postId string, uid string) (uint64, error) {
	uidint := util.AsUint(uid)
	postIdint := util.AsUint(postId)

	var exist = false
	var log = LogPostLike{Uid: uidint, PostId: postIdint}

	// 首先判断点没点过赞
	if err := db.Where(log).Find(&log).Error; err != nil {
		return 0, err
	}
	if log.Id > 0 {
		return 0, fmt.Errorf("已点赞")
	}

	if err := db.Unscoped().Where(log).Find(&log).Error; err != nil {
		return 0, err
	}

	exist = log.Id > 0
	if exist {
		if err := db.Unscoped().Model(&log).Update("deleted_at", gorm.Expr("NULL")).Error; err != nil {
			return 0, err
		}
	} else {
		if err := db.Select("uid", "post_id").Create(&log).Error; err != nil {
			return 0, err
		}
	}
	// 更新楼的likes
	var post Post
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&post).Update("like_count", post.LikeCount+1).Error; err != nil {
		return 0, err
	}
	if _, err := UnDisPost(postId, uid); err != nil {
		return 0, err
	}
	return post.LikeCount + 1, nil
}

func UnLikePost(postId string, uid string) (uint64, error) {
	uidint := util.AsUint(uid)
	postIdint := util.AsUint(postId)

	var exist = false
	var log = LogPostLike{Uid: uidint, PostId: postIdint}
	// 首先判断点没点过赞
	if err := db.Where(log).Find(&log).Error; err != nil {
		return 0, err
	}
	if log.Id == 0 {
		return 0, fmt.Errorf("未点赞")
	}

	if err := db.Where(log).Find(&log).Error; err != nil {
		return 0, err
	}
	exist = log.Id > 0
	if exist {
		if err := db.Delete(&log).Error; err != nil {
			return 0, err
		}
	}

	// 更新楼的likes
	var post Post
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&post).Update("like_count", post.LikeCount-1).Error; err != nil {
		return 0, err
	}
	return post.LikeCount - 1, nil
}

func DisPost(postId string, uid string) (uint64, error) {
	uidint := util.AsUint(uid)
	postIdint := util.AsUint(postId)

	var exist = false
	var log = LogPostDis{Uid: uidint, PostId: postIdint}

	// 首先判断点没点过赞
	if err := db.Where(log).Find(&log).Error; err != nil {
		return 0, err
	}
	if log.Id > 0 {
		return 0, fmt.Errorf("已点踩")
	}

	if err := db.Unscoped().Where(log).Find(&log).Error; err != nil {
		return 0, err
	}

	exist = log.Id > 0
	if exist {
		if err := db.Unscoped().Model(&log).Update("deleted_at", gorm.Expr("NULL")).Error; err != nil {
			return 0, err
		}
	} else {
		if err := db.Select("uid", "post_id").Create(&log).Error; err != nil {
			return 0, err
		}
	}
	// 更新楼的likes
	var post Post
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&post).Update("dis_count", post.DisCount+1).Error; err != nil {
		return 0, err
	}
	if _, err := UnLikePost(postId, uid); err != nil {
		return 0, err
	}
	return post.DisCount + 1, nil
}

func UnDisPost(postId string, uid string) (uint64, error) {
	uidint := util.AsUint(uid)
	postIdint := util.AsUint(postId)

	var exist = false
	var log = LogPostDis{Uid: uidint, PostId: postIdint}
	// 首先判断点没点过赞
	if err := db.Where(log).Find(&log).Error; err != nil {
		return 0, err
	}
	if log.Id == 0 {
		return 0, fmt.Errorf("未点踩")
	}

	if err := db.Where(log).Find(&log).Error; err != nil {
		return 0, err
	}
	exist = log.Id > 0
	if exist {
		if err := db.Delete(&log).Error; err != nil {
			return 0, err
		}
	}

	// 更新楼的likes
	var post Post
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&post).Update("dis_count", post.DisCount-1).Error; err != nil {
		return 0, err
	}
	return post.DisCount - 1, nil
}

func IsLikePostByUid(uid, postId string) bool {
	var log LogPostLike
	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Find(&log).Error; err != nil {
		logging.Error(err.Error())
		return false
	}
	return log.Id > 0
}

func IsDisPostByUid(uid, postId string) bool {
	var log LogPostDis
	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Find(&log).Error; err != nil {
		logging.Error(err.Error())
		return false
	}
	return log.Id > 0
}

func IsFavPostByUid(uid, postId string) bool {
	var log LogPostFav
	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Find(&log).Error; err != nil {
		logging.Error(err.Error())
		return false
	}
	return log.Id > 0
}

func IsOwnPostByUid(uid, postId string) bool {
	var post, err = GetPost(postId)
	if err != nil {
		return false
	}
	return fmt.Sprintf("%d", post.Uid) == uid
}

func (LogPostFav) TableName() string {
	return "log_post_fav"
}

func (LogPostLike) TableName() string {
	return "log_post_like"
}

func (LogPostDis) TableName() string {
	return "log_post_dis"
}

func (Post) TableName() string {
	return "posts"
}
