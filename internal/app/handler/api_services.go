package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"web_backend/internal/app/repository"
)

// APIGetServices возвращает список услуг (маршрутов) с фильтрацией по теме.
// GET /api/services?query=...
func (h *Handler) APIGetServices(ctx *gin.Context) {
	searchQuery := ctx.Query("query")

	routes, err := h.Repository.SearchRoutes(searchQuery)
	if err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"items": routes,
	})
}

// APIGetService возвращает одну услугу по ID.
// GET /api/services/:id
func (h *Handler) APIGetService(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	route, err := h.Repository.GetRoute(id)
	if err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusNotFound)
		return
	}

	ctx.JSON(http.StatusOK, route)
}

type createServiceRequest struct {
	Title       string  `form:"title" binding:"required"`
	From        string  `form:"from_body" binding:"required"`
	To          string  `form:"to_body" binding:"required"`
	Description string  `form:"description" binding:"required"`
	FromOrbitKm float64 `form:"from_orbit_radius_km" binding:"required"`
	ToOrbitKm   float64 `form:"to_orbit_radius_km" binding:"required"`
}

// APICreateService добавляет новую услугу и загружает файлы в Minio-совместимое хранилище.
// POST /api/services (multipart/form-data)
func (h *Handler) APICreateService(ctx *gin.Context) {
	var req createServiceRequest
	if err := ctx.Bind(&req); err != nil {
		ctx.Status(http.StatusBadRequest)
		return
	}

	imageFile, _ := ctx.FormFile("image")
	videoFile, _ := ctx.FormFile("video")

	var imageKey, videoKey string
	if imageFile != nil {
		imageKey = generateObjectKey("img", imageFile.Filename)
	}
	if videoFile != nil {
		videoKey = generateObjectKey("vid", videoFile.Filename)
	}

	// Для простоты лабораторной сохраняем файлы в локальное хранилище Minio-совместимым образом:
	// в реальном проекте здесь был бы Minio SDK / HTTP PUT в bucket `spaceobjects`.
	if imageFile != nil {
		if err := os.MkdirAll(filepath.Join("resources", "img"), 0o755); err != nil {
			logrus.Error(err)
			ctx.Status(http.StatusInternalServerError)
			return
		}
		if err := ctx.SaveUploadedFile(imageFile, filepath.Join("resources", "img", imageKey)); err != nil {
			logrus.Error(err)
			ctx.Status(http.StatusInternalServerError)
			return
		}
	}
	if videoFile != nil {
		if err := os.MkdirAll(filepath.Join("resources", "video"), 0o755); err != nil {
			logrus.Error(err)
			ctx.Status(http.StatusInternalServerError)
			return
		}
		if err := ctx.SaveUploadedFile(videoFile, filepath.Join("resources", "video", videoKey)); err != nil {
			logrus.Error(err)
			ctx.Status(http.StatusInternalServerError)
			return
		}
	}

	route := repository.TransferRoute{
		Title:       req.Title,
		From:        req.From,
		To:          req.To,
		Description: req.Description,
		IsDeleted:   false,
		Image:       imageKey,
		Video:       videoKey,
		FromOrbitKm: req.FromOrbitKm,
		ToOrbitKm:   req.ToOrbitKm,
	}

	if err := h.Repository.CreateRoute(&route); err != nil {
		logrus.Error(err)
		ctx.Status(http.StatusInternalServerError)
		return
	}

	ctx.JSON(http.StatusCreated, route)
}

func generateObjectKey(prefix, original string) string {
	ext := strings.ToLower(filepath.Ext(original))
	if ext == "" {
		ext = ".bin"
	}
	return prefix + "_" + time.Now().Format("20060102") + "_" + strings.ReplaceAll(uuid.NewString(), "-", "") + ext
}
