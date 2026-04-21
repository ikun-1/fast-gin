package rbac

import (
	"context"
	"errors"
	"fast-gin/dal/query"
	"fast-gin/middleware"
	"fast-gin/models"
	"fast-gin/service/common"
	"fast-gin/service/permission_serv"
	"fast-gin/utils/res"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required" display:"角色名称" example:"管理员"`
	Code        string `json:"code" binding:"required" display:"角色代码" example:"admin"`
	Description string `json:"description" example:"系统管理员"`
	Status      int8   `json:"status" example:"1"`
	PID         *uint  `json:"pid"`
}

type RolePermUri struct {
	ID     uint `uri:"id" binding:"required" display:"角色ID"`
	PermID uint `uri:"permID" binding:"required" display:"权限ID"`
}

type UserRoleUri struct {
	ID     uint `uri:"id" binding:"required" display:"用户ID"`
	RoleID uint `uri:"roleID" binding:"required" display:"角色ID"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name" example:"管理员"`
	Code        string `json:"code" example:"admin"`
	Description string `json:"description" example:"系统管理员"`
	Status      int8   `json:"status" default:"1" example:"1"`
	PID         *uint  `json:"pid" example:"1"`
}

type CreatePermissionRequest struct {
	Code      string `json:"code" binding:"required" display:"权限代码" example:"admin"`
	Name      string `json:"name" binding:"required" display:"权限名称" example:"管理员权限"`
	Module    string `json:"module"`
	Type      int8   `json:"type"`
	PID       uint   `json:"pid"`
	Path      string `json:"path"`
	Component string `json:"component"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sortOrder"`
}

type UpdatePermissionRequest struct {
	Name      string `json:"name"`
	Module    string `json:"module"`
	Type      int8   `json:"type"`
	PID       uint   `json:"pid"`
	Path      string `json:"path"`
	Component string `json:"component"`
	Icon      string `json:"icon"`
	SortOrder int    `json:"sortOrder"`
}

type AttachPermissionRequest struct {
	PermID uint `json:"permID" binding:"required" display:"权限ID"`
}

type AttachRoleRequest struct {
	RoleID uint `json:"roleID" binding:"required" display:"角色ID"`
}

func roleNameExists(ctx *gin.Context, name string, excludeID uint) (bool, error) {
	queryBuilder := query.Role.WithContext(ctx).Where(query.Role.Name.Eq(name))
	if excludeID != 0 {
		queryBuilder = queryBuilder.Not(query.Role.ID.Eq(excludeID))
	}
	_, err := queryBuilder.First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func roleCodeExists(ctx *gin.Context, code string, excludeID uint) (bool, error) {
	queryBuilder := query.Role.WithContext(ctx).Where(query.Role.Code.Eq(code))
	if excludeID != 0 {
		queryBuilder = queryBuilder.Not(query.Role.ID.Eq(excludeID))
	}
	_, err := queryBuilder.First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func permissionCodeExists(ctx *gin.Context, code string, excludeID uint) (bool, error) {
	queryBuilder := query.Permission.WithContext(ctx).Where(query.Permission.Code.Eq(code))
	if excludeID != 0 {
		queryBuilder = queryBuilder.Not(query.Permission.ID.Eq(excludeID))
	}
	_, err := queryBuilder.First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isRoleInheritanceCycle(roleID uint, parentID uint) bool {
	visited := map[uint]bool{roleID: true}
	cur := parentID
	for cur != 0 {
		if visited[cur] {
			return true
		}
		visited[cur] = true

		role, err := query.Role.WithContext(context.Background()).Select(query.Role.PID).Where(query.Role.ID.Eq(cur)).First()
		if err != nil {
			return false
		}
		if role.PID == nil {
			return false
		}
		cur = *role.PID
	}
	return false
}

// ListRoles 角色列表
// @Summary      角色列表
// @Description  分页查询角色，支持按名称或编码模糊搜索
// @Tags         rbac/role
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        pageInfo  query  models.PageInfo  false  "分页参数和搜索条件"
// @Success      200  {object}  res.Response
// @Router       /rbac/roles [get]
func (RBAC) ListRoles(c *gin.Context) {
	pageInfo := middleware.GetQuery[models.PageInfo](c)
	list, count, _ := common.QueryList(models.Role{}, common.QueryOption{
		PageInfo: pageInfo,
		Likes:    []string{"name", "code"},
	})
	res.OkWithList(c, list, count)
}

// GetRole 角色详情
// @Summary      角色详情
// @Description  按角色ID获取角色详情
// @Tags         rbac/role
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id  path  int  true  "角色ID"
// @Success      200  {object}  res.Response
// @Router       /rbac/roles/{id} [get]
func (RBAC) GetRole(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)

	role, err := query.Role.WithContext(c).Where(query.Role.ID.Eq(uri.ID)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			res.FailNotFound(c)
			return
		}
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkWithData(c, role)
}

// CreateRole 创建角色
// @Summary      创建角色
// @Description  创建新角色，可选指定父角色形成继承关系
// @Tags         rbac/role
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        role  body  CreateRoleRequest  true  "角色信息"
// @Success      200  {object}  res.Response
// @Router       /rbac/roles [post]
func (RBAC) CreateRole(c *gin.Context) {
	cr := middleware.GetJSON[CreateRoleRequest](c)
	if exists, err := roleNameExists(c, cr.Name, 0); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	} else if exists {
		res.FailWithMsg(c, "角色名称已存在")
		return
	}
	if exists, err := roleCodeExists(c, cr.Code, 0); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	} else if exists {
		res.FailWithMsg(c, "角色编码已存在")
		return
	}
	if cr.PID != nil {
		_, err := query.Role.WithContext(c).Where(query.Role.ID.Eq(*cr.PID)).First()
		if err != nil {
			res.FailWithMsg(c, "父角色不存在")
			return
		}
	}

	role := &models.Role{
		Name:        cr.Name,
		Code:        cr.Code,
		Description: cr.Description,
		Status:      cr.Status,
		PID:         cr.PID,
	}
	if err := query.Role.WithContext(c).Create(role); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	permission_serv.LogRolePermissionCache()

	res.OkWithData(c, role)
}

// UpdateRole 更新角色
// @Summary      更新角色
// @Description  按角色ID更新角色信息，支持修改父角色
// @Tags         rbac/role
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id    path  int                true  "角色ID"
// @Param        role  body  UpdateRoleRequest  true  "角色信息"
// @Success      200  {object}  res.Response
// @Router       /rbac/roles/{id} [put]
func (RBAC) UpdateRole(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)
	req := middleware.GetJSON[UpdateRoleRequest](c)

	role, err := query.Role.WithContext(c).Where(query.Role.ID.Eq(uri.ID)).First()
	if err != nil {
		res.FailNotFound(c)
		return
	}
	if exists, err := roleNameExists(c, req.Name, uri.ID); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	} else if exists {
		res.FailWithMsg(c, "角色名称已存在")
		return
	}
	if exists, err := roleCodeExists(c, req.Code, uri.ID); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	} else if exists {
		res.FailWithMsg(c, "角色编码已存在")
		return
	}

	if req.PID != nil {
		if *req.PID == uri.ID {
			res.FailWithMsg(c, "父角色不能是自己")
			return
		}
		_, err := query.Role.WithContext(c).Where(query.Role.ID.Eq(*req.PID)).First()
		if err != nil {
			res.FailWithMsg(c, "父角色不存在")
			return
		}
		if isRoleInheritanceCycle(uri.ID, *req.PID) {
			res.FailWithMsg(c, "角色继承关系存在环")
			return
		}
	}

	// 保存旧的ParentRoleID用于后续判断
	oldParentRoleID := role.PID

	role.Name = req.Name
	role.Code = req.Code
	role.Description = req.Description
	role.Status = req.Status
	role.PID = req.PID
	if _, err := query.Role.WithContext(c).Updates(role); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	// 仅在ParentRoleID改变时才清理缓存
	parentChanged := false
	if (oldParentRoleID == nil) != (req.PID == nil) {
		parentChanged = true
	} else if oldParentRoleID != nil && req.PID != nil && *oldParentRoleID != *req.PID {
		parentChanged = true
	}

	if parentChanged {
		_ = permission_serv.OnRoleInheritanceChanged(uri.ID)
		permission_serv.LogRolePermissionCache()
	}

	res.OkWithData(c, role)
}

// DeleteRole 删除角色
// @Summary      删除角色
// @Description  按角色ID删除角色，并清理其关联关系
// @Tags         rbac/role
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id  path  int  true  "角色ID"
// @Success      200  {object}  res.Response
// @Router       /rbac/roles/{id} [delete]
func (RBAC) DeleteRole(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)

	role, err := query.Role.WithContext(c).Where(query.Role.ID.Eq(uri.ID)).First()
	if err != nil {
		res.FailNotFound(c)
		return
	}

	// 删除前先清理该角色及相关用户缓存
	_ = permission_serv.OnRoleInheritanceChanged(uri.ID)

	// 子角色断开继承
	if _, err := query.Role.WithContext(c).
		Where(query.Role.PID.Eq(uri.ID)).
		Update(query.Role.PID, nil); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	// 删除角色-权限关联
	if _, err := query.RolePermission.WithContext(c).Where(query.RolePermission.RoleID.Eq(uri.ID)).Delete(); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}
	// 删除用户-角色关联
	if _, err := query.UserRole.WithContext(c).Where(query.UserRole.RoleID.Eq(uri.ID)).Delete(); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}
	// 删除角色
	if _, err := query.Role.WithContext(c).Delete(role); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	permission_serv.LogRolePermissionCache()
	res.OkSuccess(c)
}

// RewarmPermissionCache 重新预热权限缓存
// @Summary      重新预热权限缓存
// @Description  清空并重建全量角色与用户权限缓存
// @Tags         rbac/cache
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  res.Response
// @Router       /rbac/permission-cache/rewarm [post]
func (RBAC) RewarmPermissionCache(c *gin.Context) {
	permission_serv.RewarmAllPermCache()
	res.OkSuccess(c)
}

// ListPermissions 权限列表
// @Summary      权限列表
// @Description  分页查询权限，支持按名称或编码模糊搜索
// @Tags         rbac/permission
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        pageInfo  query  models.PageInfo  false  "分页参数和搜索条件"
// @Success      200  {object}  res.Response
// @Router       /rbac/permissions [get]
func (RBAC) ListPermissions(c *gin.Context) {
	pageInfo := middleware.GetQuery[models.PageInfo](c)
	list, count, _ := common.QueryList(models.Permission{}, common.QueryOption{
		PageInfo: pageInfo,
		Likes:    []string{"name", "code"},
	})
	res.OkWithList(c, list, count)
}

// GetPermission 权限详情
// @Summary      权限详情
// @Description  按权限ID获取权限详情
// @Tags         rbac/permission
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id  path  int  true  "权限ID"
// @Success      200  {object}  res.Response
// @Router       /rbac/permissions/{id} [get]
func (RBAC) GetPermission(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)

	perm, err := query.Permission.WithContext(c).Where(query.Permission.ID.Eq(uri.ID)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			res.FailNotFound(c)
			return
		}
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkWithData(c, perm)
}

// CreatePermission 创建权限
// @Summary      创建权限
// @Description  创建新的权限项
// @Tags         rbac/permission
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        permission  body  CreatePermissionRequest  true  "权限信息"
// @Success      200  {object}  res.Response
// @Router       /rbac/permissions [post]
func (RBAC) CreatePermission(c *gin.Context) {
	req := middleware.GetJSON[CreatePermissionRequest](c)
	if req.Type < 1 || req.Type > 3 {
		req.Type = 2
	}
	if exists, err := permissionCodeExists(c, req.Code, 0); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	} else if exists {
		res.FailWithMsg(c, "权限编码已存在")
		return
	}
	perm := &models.Permission{
		Code:      req.Code,
		Name:      req.Name,
		Module:    req.Module,
		Type:      req.Type,
		PID:       req.PID,
		Path:      req.Path,
		Component: req.Component,
		Icon:      req.Icon,
		SortOrder: req.SortOrder,
	}
	if err := query.Permission.WithContext(c).Create(perm); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}
	res.OkWithData(c, perm)
}

// UpdatePermission 更新权限
// @Summary      更新权限
// @Description  按权限ID更新权限信息
// @Tags         rbac/permission
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id          path  int                      true  "权限ID"
// @Param        permission  body  UpdatePermissionRequest  true  "权限信息"
// @Success      200  {object}  res.Response
// @Router       /rbac/permissions/{id} [put]
func (RBAC) UpdatePermission(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)
	req := middleware.GetJSON[UpdatePermissionRequest](c)
	if req.Type < 1 || req.Type > 3 {
		req.Type = 2
	}

	perm, err := query.Permission.WithContext(c).Where(query.Permission.ID.Eq(uri.ID)).First()
	if err != nil {
		res.FailNotFound(c)
		return
	}

	perm.Name = req.Name
	perm.Module = req.Module
	perm.Type = req.Type
	perm.PID = req.PID
	perm.Path = req.Path
	perm.Component = req.Component
	perm.Icon = req.Icon
	perm.SortOrder = req.SortOrder
	if _, err := query.Permission.WithContext(c).Updates(perm); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkWithData(c, perm)
}

// DeletePermission 删除权限
// @Summary      删除权限
// @Description  按权限ID删除权限，并清理角色-权限关联
// @Tags         rbac/permission
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id  path  int  true  "权限ID"
// @Success      200  {object}  res.Response
// @Router       /rbac/permissions/{id} [delete]
func (RBAC) DeletePermission(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)

	perm, err := query.Permission.WithContext(c).Where(query.Permission.ID.Eq(uri.ID)).First()
	if err != nil {
		res.FailNotFound(c)
		return
	}

	var roleIDs []uint
	err = query.RolePermission.WithContext(c).Where(query.RolePermission.PermID.Eq(uri.ID)).Pluck(query.RolePermission.RoleID, &roleIDs)
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	if _, err := query.RolePermission.WithContext(c).Where(query.RolePermission.PermID.Eq(uri.ID)).Delete(); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}
	if _, err := query.Permission.WithContext(c).Delete(perm); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	for _, id := range roleIDs {
		_ = permission_serv.OnRolePermissionChanged(id)
	}
	permission_serv.LogRolePermissionCache()
	res.OkSuccess(c)
}

// ListRolePermissions 角色权限列表
// @Summary      角色权限列表
// @Description  查询某角色绑定的权限列表
// @Tags         rbac/role-permission
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id  path  int  true  "角色ID"
// @Success      200  {object}  res.Response
// @Router       /rbac/roles/{id}/permissions [get]
func (RBAC) ListRolePermissions(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)

	list, err := query.Permission.WithContext(c).
		Join(query.RolePermission, query.Permission.ID.EqCol(query.RolePermission.PermID)).
		Where(query.RolePermission.RoleID.Eq(uri.ID)).
		Order(query.Permission.ID.Desc()).
		Find()
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkWithData(c, list)
}

// AttachRolePermission 绑定角色权限
// @Summary      绑定角色权限
// @Description  为角色绑定一个权限
// @Tags         rbac/role-permission
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id   path  int                      true  "角色ID"
// @Param        body body  AttachPermissionRequest  true  "权限绑定信息"
// @Success      200  {object}  res.Response
// @Router       /rbac/roles/{id}/permissions [post]
func (RBAC) AttachRolePermission(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)
	req := middleware.GetJSON[AttachPermissionRequest](c)

	_, err := query.Role.WithContext(c).Where(query.Role.ID.Eq(uri.ID)).First()
	if err != nil {
		res.FailWithMsg(c, "角色不存在")
		return
	}
	_, err = query.Permission.WithContext(c).Where(query.Permission.ID.Eq(req.PermID)).First()
	if err != nil {
		res.FailWithMsg(c, "权限不存在")
		return
	}

	_, err = query.RolePermission.WithContext(c).
		Where(query.RolePermission.RoleID.Eq(uri.ID), query.RolePermission.PermID.Eq(req.PermID)).
		FirstOrCreate()
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	_ = permission_serv.OnRolePermissionChanged(uri.ID)
	permission_serv.LogRolePermissionCache()
	res.OkSuccess(c)
}

// DetachRolePermission 解绑角色权限
// @Summary      解绑角色权限
// @Description  删除角色与权限的绑定关系
// @Tags         rbac/role-permission
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id      path  int  true  "角色ID"
// @Param        permID  path  int  true  "权限ID"
// @Success      200  {object}  res.Response
// @Router       /rbac/roles/{id}/permissions/{permID} [delete]
func (RBAC) DetachRolePermission(c *gin.Context) {
	uri := middleware.GetUri[RolePermUri](c)

	if _, err := query.RolePermission.WithContext(c).
		Where(query.RolePermission.RoleID.Eq(uri.ID), query.RolePermission.PermID.Eq(uri.PermID)).
		Delete(); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}
	_ = permission_serv.OnRolePermissionChanged(uri.ID)
	permission_serv.LogRolePermissionCache()
	res.OkSuccess(c)
}

// ListUserRoles 用户角色列表
// @Summary      用户角色列表
// @Description  查询某用户绑定的角色列表
// @Tags         rbac/user-role
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id  path  int  true  "用户ID"
// @Success      200  {object}  res.Response
// @Router       /rbac/users/{id}/roles [get]
func (RBAC) ListUserRoles(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)

	list, err := query.Role.WithContext(c).
		Join(query.UserRole, query.Role.ID.EqCol(query.UserRole.RoleID)).
		Where(query.UserRole.UserID.Eq(uri.ID)).
		Order(query.Role.ID.Desc()).
		Find()
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	res.OkWithData(c, list)
}

// AttachUserRole 绑定用户角色
// @Summary      绑定用户角色
// @Description  为用户绑定一个角色
// @Tags         rbac/user-role
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id    path  int                true  "用户ID"
// @Param        body  body  AttachRoleRequest  true  "角色绑定信息"
// @Success      200  {object}  res.Response
// @Router       /rbac/users/{id}/roles [post]
func (RBAC) AttachUserRole(c *gin.Context) {
	uri := middleware.GetUri[models.BindId](c)
	req := middleware.GetJSON[AttachRoleRequest](c)

	_, err := query.User.WithContext(c).Where(query.User.ID.Eq(uri.ID)).First()
	if err != nil {
		res.FailWithMsg(c, "用户不存在")
		return
	}
	_, err = query.Role.WithContext(c).Where(query.Role.ID.Eq(req.RoleID)).First()
	if err != nil {
		res.FailWithMsg(c, "角色不存在")
		return
	}

	_, err = query.UserRole.WithContext(c).
		Where(query.UserRole.UserID.Eq(uri.ID), query.UserRole.RoleID.Eq(req.RoleID)).
		FirstOrCreate()
	if err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}

	_ = permission_serv.OnUserRoleChanged(uri.ID)
	res.OkSuccess(c)
}

// DetachUserRole 解绑用户角色
// @Summary      解绑用户角色
// @Description  删除用户与角色的绑定关系
// @Tags         rbac/user-role
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        id      path  int  true  "用户ID"
// @Param        roleID  path  int  true  "角色ID"
// @Success      200  {object}  res.Response
// @Router       /rbac/users/{id}/roles/{roleID} [delete]
func (RBAC) DetachUserRole(c *gin.Context) {
	uri := middleware.GetUri[UserRoleUri](c)

	if _, err := query.UserRole.WithContext(c).
		Where(query.UserRole.UserID.Eq(uri.ID), query.UserRole.RoleID.Eq(uri.RoleID)).
		Delete(); err != nil {
		res.FailWithCode(c, res.DatabaseErr)
		return
	}
	_ = permission_serv.OnUserRoleChanged(uri.ID)
	res.OkSuccess(c)
}
