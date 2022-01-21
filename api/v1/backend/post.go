package backend

import (
	"qnhd/models"
	"qnhd/pkg/e"
	"qnhd/pkg/logging"
	"qnhd/pkg/r"
	"qnhd/pkg/util"

	"github.com/astaxie/beego/validation"
	"github.com/gin-gonic/gin"
)

// @method [get]
// @way [query]
// @param content page page_size
// @return postList
// @route /b/posts
func GetPosts(c *gin.Context) {
	postType := c.Query("type")
	content := c.Query("content")
	departmentId := c.Query("department_id")
	solved := c.Query("solved")
	tagId := c.Query("tag_id")

	valid := validation.Validation{}
	valid.Required(postType, "type")
	valid.Numeric(postType, "type")
	if solved != "" {
		valid.Numeric(solved, "solved")
	}
	if departmentId != "" {
		valid.Numeric(departmentId, "department_id")
	}
	if tagId != "" {
		valid.Numeric(tagId, "tag_id")
	}
	ok, verr := r.ErrorValid(&valid, "Get posts")
	if !ok {
		r.OK(c, e.INVALID_PARAMS, map[string]interface{}{"error": verr.Error()})
		return
	}
	postTypeint := util.AsInt(postType)
	valid.Range(postTypeint, 0, 2, "postType")
	if solved != "" {
		solvedint := util.AsInt(solved)
		valid.Range(solvedint, 0, 1, "solved")
	}
	ok, verr = r.ErrorValid(&valid, "Get posts")
	if !ok {
		r.OK(c, e.INVALID_PARAMS, map[string]interface{}{"error": verr.Error()})
		return
	}

	maps := map[string]interface{}{
		"type":          models.PostType(postTypeint),
		"content":       content,
		"solved":        solved,
		"department_id": departmentId,
		"tag_id":        tagId,
	}

	list, err := models.GetPosts(c, maps)
	if err != nil {
		logging.Error("Get posts error: %v", err)
		r.Error(c, e.ERROR_DATABASE, err.Error())
		return
	}

	data := make(map[string]interface{})
	data["list"] = list
	data["total"] = len(list)

	r.OK(c, e.SUCCESS, data)
}

// @method [post]
// @way [formdata]
// @param id
// @return post
// @route /b/post
func GetPost(c *gin.Context) {
	id := c.Query("id")
	valid := validation.Validation{}
	valid.Required(id, "id")
	valid.Numeric(id, "id")

	ok, verr := r.ErrorValid(&valid, "Get Posts")
	if !ok {
		r.OK(c, e.INVALID_PARAMS, map[string]interface{}{"error": verr.Error()})
		return
	}

	pr, err := models.GetPost(id)
	if err != nil {
		logging.Error("Get post error: %v", err)
		r.Error(c, e.ERROR_DATABASE, err.Error())
		return
	}
	data := map[string]interface{}{
		"post": pr,
	}
	r.OK(c, e.SUCCESS, data)
}

// @method [post]
// @way [query]
// @param
// @return
// @route
// @method [delete]
// @way [query]
// @param id
// @return
// @route /b/post/delete
func DeletePosts(c *gin.Context) {
	uid := r.GetUid(c)
	id := c.Query("id")

	valid := validation.Validation{}
	valid.Required(id, "id")
	valid.Numeric(id, "id")
	ok, verr := r.ErrorValid(&valid, "Delete post")
	if !ok {
		r.OK(c, e.INVALID_PARAMS, map[string]interface{}{"error": verr.Error()})
		return
	}

	_, err := models.DeletePostsAdmin(uid, id)
	if err != nil {
		logging.Error("Delete posts error: %v", err)
		r.Error(c, e.ERROR_DATABASE, err.Error())
		return
	}
	r.OK(c, e.SUCCESS, nil)
}
