package ManagerLogType

var msgSymbol = map[Enum]string{
	USER_BAN:   "user_ban",
	USER_UNBAN: "user_unban",

	USER_BLOCK:   "user_block",
	USER_UNBLOCK: "user_unblock",

	POST_DELETE:  "post_delete",
	FLOOR_DELETE: "floor_delete",

	POST_ETAG:   "post_etag",
	POST_UNETAG: "post_unetag",

	POST_TOP:   "post_top",
	POST_UNTOP: "post_untop",

	POST_REPLY:               "post_reply",
	POST_DEPARTMENT_TRANSFER: "post_department_transfer",
	POST_TPYE_TRANSFER:       "post_tpye_transfer",

	USER_ADD:               "user_add",
	USER_PERMISSION_CHANGE: "user_permission_change",

	NOTICE_NEW:    "notice_new",
	NOTICE_DELETE: "notice_delete",
	NOTICE_CHANGE: "notice_change",

	USER_DETAIL: "user_detail",
}

func (code Enum) GetSymbol() string {
	return msgSymbol[code]
}
