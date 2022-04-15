package models

import (
	"errors"
	"fmt"
	"qnhd/pkg/filter"
	"qnhd/pkg/logging"
	"qnhd/pkg/segment"

	"qnhd/pkg/util"

	"github.com/gin-gonic/gin"
	giterrors "github.com/pkg/errors"
	"gorm.io/gorm"
)

const POST_ALL = 0

type PostCampusType int

const (
	CAMPUS_NONE PostCampusType = 0
	CAMPUS_OLD  PostCampusType = 1
	CAMPUS_NEW  PostCampusType = 2
)

type PostSearchModeType int

const (
	SEARCH_BY_TIME   PostSearchModeType = 0
	SEARCH_BY_UPDATE PostSearchModeType = 1
)

type PostValueModeType int

const (
	VALUE_DEFAULT PostValueModeType = 0
	VALUE_NONE    PostValueModeType = 1
	VALUE_ONLY    PostValueModeType = 2
)

type PostSolveType int

const (
	POST_UNSOLVED PostSolveType = 0
	POST_REPLIED  PostSolveType = 1
	POST_SOLVED   PostSolveType = 2
)

type Post struct {
	Model
	Uid uint64 `json:"uid" gorm:"column:uid"`

	// 帖子分类
	Type         int            `json:"type"`
	DepartmentId uint64         `json:"-" gorm:"column:department_id;default:0"`
	Campus       PostCampusType `json:"campus"`
	Solved       PostSolveType  `json:"solved" gorm:"default:0"`

	// 帖子内容
	Title   string `json:"title"`
	Content string `json:"content"`

	// 各种数量
	FavCount  uint64 `json:"fav_count" gorm:"default:0"`
	LikeCount uint64 `json:"like_count" gorm:"default:0"`
	DisCount  uint64 `json:"-" gorm:"default:0"`

	// 评分
	Rating uint64 `json:"rating" gorm:"default:0"`
	// 加精值
	Value uint64 `json:"value" gorm:"default:0"`

	// 分词
	Tokens string `json:"-"`

	UpdatedAt string `json:"-" gorm:"default:null;"`
}

type LogPostFav struct {
	Uid    uint64 `json:"uid"`
	PostId uint64 `json:"post_id"`
}
type LogPostLike struct {
	Uid    uint64 `json:"uid"`
	PostId uint64 `json:"post_id"`
}
type LogPostDis struct {
	Uid    uint64 `json:"uid"`
	PostId uint64 `json:"post_id"`
}

// 帖子返回数据
type PostResponse struct {
	Post
	Tag          *Tag            `json:"tag"`
	Floors       []FloorResponse `json:"floors"`
	CommentCount int             `json:"comment_count"`
	ImageUrls    []string        `json:"image_urls"`
	Department   *Department     `json:"department"`
	IsDeleted    bool            `json:"is_deleted"`
	// 用于处理链式数据
	Error error `json:"-"`
}

// 客户端帖子返回数据
type PostResponseUser struct {
	Post
	Tag          *Tag                `json:"tag"`
	Floors       []FloorResponseUser `json:"floors"`
	CommentCount int                 `json:"comment_count"`
	ImageUrls    []string            `json:"image_urls"`
	Department   *Department         `json:"department"`

	IsLike    bool `json:"is_like"`
	IsDis     bool `json:"is_dis"`
	IsFav     bool `json:"is_fav"`
	IsOwner   bool `json:"is_owner"`
	IsDeleted bool `json:"is_deleted"`
	// 用于处理链式数据
	Error error `json:"-"`
}

func (p *Post) geneResponse(unscoped bool) PostResponse {
	var pr PostResponse

	// frs, err := getShortFloorResponsesInPost(util.AsStrU(p.Id))
	// if err != nil {
	// 	pr.Error = err
	// 	return pr
	// }
	imgs, err := GetImageInPost(p.Id)
	if err != nil {
		pr.Error = err
		return pr
	}
	pr = PostResponse{
		Post:         *p,
		CommentCount: GetCommentCount(p.Id, true, unscoped),
		ImageUrls:    imgs,
	}

	if p.DepartmentId > 0 {
		d, err := GetDepartment(p.DepartmentId)
		if err != nil {
			pr.Error = err
			return pr
		}
		pr.Department = &d
	}
	tag, _ := GetTagInPost(util.AsStrU(p.Id))
	if tag != nil {
		pr.Tag = tag
	}
	pr.Error = err
	pr.IsDeleted = pr.DeletedAt.Valid
	return pr
}

func (p PostResponse) searchByUid(uid string) PostResponseUser {
	pr := PostResponseUser{
		Post:         p.Post,
		Tag:          p.Tag,
		CommentCount: p.CommentCount,
		ImageUrls:    p.ImageUrls,
		Department:   p.Department,
		IsLike:       IsLikePostByUid(uid, util.AsStrU(p.Id)),
		IsDis:        IsDisPostByUid(uid, util.AsStrU(p.Id)),
		IsFav:        IsFavPostByUid(uid, util.AsStrU(p.Id)),
		IsOwner:      IsOwnPostByUid(uid, util.AsStrU(p.Id)),
	}

	// frs, err := getShortFloorResponsesInPostWithUid(util.AsStrU(p.Id), uid)
	// if err != nil {
	// 	pr.Error = err
	// 	return pr
	// }
	// pr.Floors = frs
	return pr
}

// 将post数组转化为返回结果，后台使用
func transPostsToResponses(posts *[]Post) ([]PostResponse, error) {
	var prs = []PostResponse{}
	var err error
	for _, p := range *posts {
		pr := p.geneResponse(true)
		if pr.Error != nil {
			err = giterrors.Wrap(err, pr.Error.Error())
		} else {
			prs = append(prs, pr)
		}
	}
	return prs, err
}

// 将post数组转化为用户返回结果， 前端使用
func transPostsToResponsesWithUid(posts *[]Post, uid string) ([]PostResponseUser, error) {
	var prs = []PostResponseUser{}
	var err error
	for _, p := range *posts {
		pr := p.geneResponse(false).searchByUid(uid)
		if pr.Error != nil {
			err = giterrors.Wrap(err, pr.Error.Error())
		} else {
			prs = append(prs, pr)
		}
	}
	return prs, err
}

func GetPost(postId string) (Post, error) {
	var post Post
	err := db.Where("id = ?", postId).First(&post).Error
	return post, err
}

// 后台使用
func GetPostResponse(postId string) (PostResponse, error) {
	var p Post
	var pr PostResponse
	err := db.Unscoped().Where("id = ?", postId).First(&p).Error
	if err != nil {
		return pr, err
	}
	pr = p.geneResponse(true)
	return pr, pr.Error
}

// 前端使用
func GetPostResponseUserAndVisit(postId string, uid string) (PostResponseUser, error) {
	var post Post
	var pr PostResponseUser
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		return pr, err
	}
	if err := AddVisitHistory(uid, postId); err != nil {
		return pr, err
	}
	ret := post.geneResponse(false).searchByUid(uid)
	return ret, ret.Error
}

// front表示是否为前端请求
func getPosts(c *gin.Context, maps map[string]interface{}) ([]Post, int, error) {
	var (
		posts []Post
		cnt   int64
		err   error
	)
	content := maps["content"].(string)
	postType := maps["type"].(int)
	searchMode := maps["search_mode"].(PostSearchModeType)
	departmentId := maps["department_id"].(string)
	solved := maps["solved"].(string)
	tagId := maps["tag_id"].(string)
	valueMode := maps["value_mode"].(PostValueModeType)
	front := maps["front"].(bool)

	var d = db.Model(&Post{})
	// 如果是前端
	if !front {
		d = d.Unscoped()
	}
	// 加精帖搜索
	if valueMode == VALUE_DEFAULT {
		d = d.Order("value DESC")
	} else if valueMode == VALUE_ONLY {
		d = d.Where("value <> 0")
	} else if valueMode == VALUE_NONE {
		// VALUE_NONE 不做操作
	}

	// 当搜索不为空时加上全文检索
	if content != "" {
		d = db.Select("p.*", "ts_rank(p.tokens, q) as score").
			Table("(?) as p, plainto_tsquery(?) as q", d, segment.Cut(content, " ")).
			Where("q @@ p.tokens").Order("score DESC")
	}
	// 排序方式
	if searchMode == SEARCH_BY_TIME {
		d = d.Order("created_at DESC")
	} else if searchMode == SEARCH_BY_UPDATE {
		d = d.Order("updated_at DESC")
	}

	// 校区 不为全部时加上区分
	if postType != POST_ALL {
		d = d.Where("type = ?", postType)
	}
	// 如果有部门要加上
	if departmentId != "" {
		d = d.Where("department_id = ?", departmentId)
	}
	// 如果要加上是否解决的字段
	if solved != "" {
		d = d.Where("solved = ?", solved)
	}
	// 如果需要搜索标签
	if tagId != "" {
		// 搜索相关帖子
		var tagIds = []uint64{}
		// 不需要处理错误，空的返回也行
		db.Model(&PostTag{}).Select("post_id").Where("tag_id = ?", tagId).Find(&tagIds)
		// 然后加上条件
		d = d.Where("id IN (?)", tagIds)
	}
	if err = d.Count(&cnt).Error; err != nil {
		return posts, int(cnt), err
	}

	err = d.Scopes(util.Paginate(c)).Find(&posts).Error
	return posts, int(cnt), err
}

// 获取帖子返回数据，后台使用
func GetPostResponses(c *gin.Context, maps map[string]interface{}) ([]PostResponse, int, error) {
	maps["front"] = false
	posts, cnt, err := getPosts(c, maps)
	if err != nil {
		return nil, 0, err
	}
	ret, err := transPostsToResponses(&posts)
	return ret, cnt, err
}

// 获取帖子返回数据带uid，前端使用
func GetPostResponsesWithUid(c *gin.Context, uid string, maps map[string]interface{}) ([]PostResponseUser, error) {
	maps["front"] = true
	posts, _, err := getPosts(c, maps)
	if err != nil {
		return nil, err
	}
	return transPostsToResponsesWithUid(&posts, uid)
}

func GetUserPostResponseUsers(c *gin.Context, uid string) ([]PostResponseUser, error) {
	var posts []Post
	if err := db.Where("uid = ?", uid).Scopes(util.Paginate(c)).Order("id DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return transPostsToResponsesWithUid(&posts, uid)
}

func GetFavPostResponseUsers(c *gin.Context, uid string) ([]PostResponseUser, error) {
	var posts []Post
	if err := db.Joins(`JOIN qnhd.log_post_fav
	ON qnhd.post.id = qnhd.log_post_fav.post_id
	AND qnhd.log_post_fav.uid = ?`, uid).Scopes(util.Paginate(c)).Order("id DESC").Find(&posts).Error; err != nil {
		return nil, err
	}
	return transPostsToResponsesWithUid(&posts, uid)
}

func GetHistoryPostResponseUsers(c *gin.Context, uid string) ([]PostResponseUser, error) {
	var posts []Post
	var ids []string
	if err := db.Model(&LogVisitHistory{}).Where("uid = ?", uid).Order("created_at DESC").Distinct("post_id").Scopes(util.Paginate(c)).Find(&ids).Error; err != nil {
		return nil, err
	}

	if err := db.Where("id IN (?)", ids).Scopes(util.Paginate(c)).Find(&posts).Error; err != nil {
		return nil, err
	}
	return transPostsToResponsesWithUid(&posts, uid)
}

func AddPost(maps map[string]interface{}) (uint64, error) {
	var err error
	var post = &Post{
		Type:    maps["type"].(int),
		Uid:     maps["uid"].(uint64),
		Campus:  maps["campus"].(PostCampusType),
		Title:   filter.Filter(maps["title"].(string)),
		Content: filter.Filter(maps["content"].(string)),
	}
	if post.Type == POST_SCHOOL_TYPE {
		// 先对department_id进行查找，不存在要报错
		departId := maps["department_id"].(uint64)
		if err = db.Where("id = ?", departId).First(&Department{}).Error; err != nil {
			return 0, err
		}
		post.DepartmentId = departId
		imgs, img_ok := maps["image_urls"].([]string)
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(post).Error; err != nil {
				return err
			}

			if img_ok {
				if err := AddImageInPost(tx, post.Id, imgs); err != nil {
					return err
				}
			}
			return nil
		})
	} else if IsValidPostType(post.Type) {
		imgs, img_ok := maps["image_urls"].([]string)
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(post).Error; err != nil {
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
				if err := AddPostWithTag(tx, post.Id, util.AsUint(tagId)); err != nil {
					return err
				}
				// 对帖子的tag增加记录
				addTagLog(util.AsUint(tagId), TAG_ADD_POST)
			}
			return nil
		})
	} else {
		return 0, fmt.Errorf("invalid post type")
	}
	if err != nil {
		return 0, err
	}
	if err := flushPostTokens(post.Id, post.Title, post.Content); err != nil {
		return 0, err
	}

	return post.Id, nil
}

func EditPost(postId string, maps map[string]interface{}) error {
	return db.Model(&Post{}).Where("id = ?", postId).Updates(maps).Error
}

func EditPostDepartment(postId string, departmentId string) error {
	// 判断是否存在部门
	var depart Department
	if err := db.First(&depart, departmentId).Error; err != nil {
		return err
	}

	return EditPost(postId, map[string]interface{}{"department_id": departmentId})
}

func EditPostType(postId string, typeId string) error {
	// 判断是否存在类型
	var pt PostType
	if err := db.First(&pt, typeId).Error; err != nil {
		return err
	}
	post, err := GetPost(postId)
	if err != nil {
		return err
	}
	// 如果类型相同
	if post.Type == util.AsInt(typeId) {
		return fmt.Errorf("不能修改为同类型")
	}
	// 如果要修改为校务类型，禁止操作
	if util.AsInt(typeId) == int(POST_SCHOOL_TYPE) {
		return fmt.Errorf("不能修改为校务类型")
	}
	// 如果是校务类型，需要去掉部门
	if post.Type == POST_SCHOOL_TYPE {
		return EditPost(postId, map[string]interface{}{"type": typeId, "department_id": 0})
	}
	return EditPost(postId, map[string]interface{}{"type": typeId})
}

func DeletePostsUser(id, uid string) (uint64, error) {
	var post = Post{}
	if err := db.Where("id = ? AND uid = ?", id, uid).First(&post).Error; err != nil {
		return 0, err
	}
	err := deletePost(&post)
	return post.Id, err
}

func DeletePostsAdmin(uid, postId string) (uint64, error) {
	var post, _ = GetPost(postId)
	err := deletePost(&post)
	return post.Id, err
}

// 删除帖子记录
func deletePost(post *Post) error {
	/*
		需要删除的内容
		reports
		post_reply
		floors
	*/
	return db.Transaction(func(tx *gorm.DB) error {
		if err := deleteReports(tx, map[string]interface{}{"post_id": post.Id}); err != nil {
			return err
		}
		// 删除log
		if err := DeletePostReplysInPost(tx, post.Id); err != nil {
			return err
		}
		if err := DeleteFloorsInPost(tx, post.Id); err != nil {
			return err
		}
		if err := tx.Delete(&post, post.Id).Error; err != nil {
			return err
		}
		return nil
	})

}

// 恢复帖子记录
func RecoverPost(postId string) error {
	/*
		需要恢复的内容
		reports
		post_reply
		floors
	*/
	return db.Transaction(func(tx *gorm.DB) error {
		var post Post
		if err := tx.Unscoped().Where("id = ?", postId).Find(&post).Error; err != nil {
			return err
		}
		if err := recoverReports(tx, map[string]interface{}{"post_id": post.Id}); err != nil {
			return err
		}
		// 删除log
		if err := RecoverPostReplysInPost(tx, post.Id); err != nil {
			return err
		}
		if err := RecoverFloorsInPost(tx, post.Id); err != nil {
			return err
		}
		if err := tx.Unscoped().Model(&Post{}).Where("id = ?", post.Id).Update("deleted_at", gorm.Expr("NULL")).Error; err != nil {
			return err
		}
		return nil
	})
}

func FavPost(postId string, uid string) (uint64, error) {
	var log LogPostFav

	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Find(&log).Error; err != nil {
		return 0, err
	}
	if log.Uid > 0 {
		return 0, fmt.Errorf("已收藏")
	}

	log.Uid = util.AsUint(uid)
	log.PostId = util.AsUint(postId)
	if err := db.Create(&log).Error; err != nil {
		return 0, err
	}
	// 更新收藏数
	var post Post
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&post).Update("fav_count", post.FavCount+1).Error; err != nil {
		return 0, err
	}

	if uid != util.AsStrU(post.Uid) {
		updatePostTime(post.Id)
		addTagLogInPost(post.Id, TAG_FAV_POST)
	}
	return post.FavCount, nil
}

func UnfavPost(postId string, uid string) (uint64, error) {
	var log LogPostFav

	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Find(&log).Error; err != nil {
		return 0, err
	}
	if log.Uid == 0 {
		return 0, fmt.Errorf("未收藏")
	}

	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Delete(&log).Error; err != nil {
		return 0, err
	}

	// 更新收藏数
	var post Post
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&post).Update("fav_count", post.FavCount-1).Error; err != nil {
		return 0, err
	}
	if uid != util.AsStrU(post.Uid) {
		addTagLogInPost(post.Id, TAG_UNFAV_POST)
	}
	return post.FavCount, nil
}

func LikePost(postId string, uid string) (uint64, error) {
	var log LogPostLike

	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Find(&log).Error; err != nil {
		return 0, err
	}

	if log.Uid > 0 {
		return 0, fmt.Errorf("已点赞")
	}
	log.Uid = util.AsUint(uid)
	log.PostId = util.AsUint(postId)
	if err := db.Create(&log).Error; err != nil {
		return 0, err
	}
	// 更新点赞数
	var post Post
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&post).Update("like_count", post.LikeCount+1).Error; err != nil {
		return 0, err
	}

	if uid != util.AsStrU(post.Uid) {
		updatePostTime(post.Id)
		addTagLogInPost(post.Id, TAG_LIKE_POST)
	}
	addUnreadLike(post.Uid, LIKE_POST, post.Id)
	UnDisPost(postId, uid)
	return post.LikeCount, nil
}

func UnLikePost(postId string, uid string) (uint64, error) {
	var log LogPostLike

	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Find(&log).Error; err != nil {
		return 0, err
	}

	if log.Uid == 0 {
		return 0, fmt.Errorf("未点赞")
	}

	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Delete(&log).Error; err != nil {
		return 0, err
	}

	// 更新点赞数
	var post Post
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&post).Update("like_count", post.LikeCount-1).Error; err != nil {
		return 0, err
	}
	if uid != util.AsStrU(post.Uid) {
		addTagLogInPost(post.Id, TAG_UNLIKE_POST)
	}
	return post.LikeCount, nil
}

func DisPost(postId string, uid string) (uint64, error) {
	var log LogPostDis

	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Find(&log).Error; err != nil {
		return 0, err
	}
	if log.Uid > 0 {
		return 0, fmt.Errorf("已点踩")
	}
	log.Uid = util.AsUint(uid)
	log.PostId = util.AsUint(postId)
	if err := db.Create(&log).Error; err != nil {
		return 0, err
	}
	// 更新点踩数
	var post Post
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&post).Update("dis_count", post.DisCount+1).Error; err != nil {
		return 0, err
	}
	if uid != util.AsStrU(post.Uid) {
		updatePostTime(post.Id)
		addTagLogInPost(post.Id, TAG_DIS_POST)
	}
	UnLikePost(postId, uid)
	return post.DisCount, nil
}

func UnDisPost(postId string, uid string) (uint64, error) {
	var log LogPostDis

	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Find(&log).Error; err != nil {
		return 0, err
	}
	if log.Uid == 0 {
		return 0, fmt.Errorf("未点踩")
	}

	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Delete(&log).Error; err != nil {
		return 0, err
	}

	// 更新楼的点踩数
	var post Post
	if err := db.Where("id = ?", postId).First(&post).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}
	}
	if err := db.Model(&post).Update("dis_count", post.DisCount-1).Error; err != nil {
		return 0, err
	}
	if uid != util.AsStrU(post.Uid) {
		addTagLogInPost(post.Id, TAG_UNDIS_POST)
	}
	return post.DisCount, nil
}

func IsLikePostByUid(uid, postId string) bool {
	var log LogPostLike
	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Find(&log).Error; err != nil {
		logging.Error(err.Error())
		return false
	}
	return log.Uid > 0
}

func IsDisPostByUid(uid, postId string) bool {
	var log LogPostDis
	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Find(&log).Error; err != nil {
		logging.Error(err.Error())
		return false
	}
	return log.Uid > 0
}

func IsFavPostByUid(uid, postId string) bool {
	var log LogPostFav
	if err := db.Where("uid = ? AND post_id = ?", uid, postId).Find(&log).Error; err != nil {
		logging.Error(err.Error())
		return false
	}
	return log.Uid > 0
}

func IsOwnPostByUid(uid, postId string) bool {
	var post, err = GetPost(postId)
	if err != nil {
		return false
	}
	return util.AsStrU(post.Uid) == uid
}

func updatePostTime(postId uint64) error {
	return db.Model(&Post{}).Where("id = ?", postId).Update("updated_at", gorm.Expr("CURRENT_TIMESTAMP")).Error
}
