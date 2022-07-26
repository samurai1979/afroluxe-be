package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/afroluxe/afroluxe-be/db"
	"github.com/afroluxe/afroluxe-be/dtos"
	"github.com/afroluxe/afroluxe-be/models"
	"github.com/afroluxe/afroluxe-be/services"
	"github.com/afroluxe/afroluxe-be/utils"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	stylistCollection  = db.CollectionInstance("stylists")
	reviewCollection   = db.CollectionInstance("reviews")
	servicesCollection = db.CollectionInstance("services")
	imagesCollection   = db.CollectionInstance("images")
)

// GetStylist ... Gets stylist
// @Summary Gets stylist based on stylist id
// @Description Verifies user email
// @Tags Stylist
// @Accept json
// @Param        id   path      string  true  "Stylist ID"
// @Success 200 {object} dtos.Response
// @Failure 400,500 {object} dtos.Response
// @Router /stylist/{id} [get]
func GetStylist(c *gin.Context) {
	stylistId := c.Param("id")
	var result models.Stylist

	mongoStylistId, err := primitive.ObjectIDFromHex(stylistId)
	if err != nil {
		utils.ErrorLogger(err)
		c.JSON(http.StatusInternalServerError, dtos.Response{
			StatusCode: http.StatusInternalServerError,
			Message:    "internal server error",
			Data:       nil,
		})
		return
	}

	err = stylistCollection.FindOne(context.TODO(), bson.D{{Key: "_id", Value: mongoStylistId}}).Decode(&result)
	if err != nil {
		c.JSON(http.StatusNotFound, dtos.Response{
			StatusCode: http.StatusNotFound,
			Message:    "Stylist not found",
			Data:       nil,
		})
		return
	}

	cursor, err := servicesCollection.Find(context.TODO(), bson.D{{Key: "stylist_id", Value: mongoStylistId}})
	if err != nil {
		utils.ErrorLogger(err)
		c.JSON(http.StatusInternalServerError, dtos.Response{
			StatusCode: http.StatusInternalServerError,
			Message:    "internal server error",
			Data:       nil,
		})
		return
	}
	err = cursor.All(context.TODO(), &result.Services)
	if err != nil {
		utils.ErrorLogger(err)
		c.JSON(http.StatusInternalServerError, dtos.Response{
			StatusCode: http.StatusInternalServerError,
			Message:    "internal server error",
			Data:       nil,
		})
		return
	}
	cursor, err = imagesCollection.Find(context.TODO(), bson.D{{Key: "stylist_id", Value: result.Id}})
	if err != nil {
		utils.ErrorLogger(err)
		c.JSON(http.StatusInternalServerError, dtos.Response{
			StatusCode: http.StatusInternalServerError,
			Message:    "internal server error",
			Data:       nil,
		})
		return
	}
	err = cursor.All(context.TODO(), &result.Images)
	if err != nil {
		utils.ErrorLogger(err)
		c.JSON(http.StatusInternalServerError, dtos.Response{
			StatusCode: http.StatusInternalServerError,
			Message:    "internal server error",
			Data:       nil,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

// CreateStylist ... Creates new stylist
// @Summary Creates new stylist
// @Description Creates new stylist
// @Tags Stylist
// @Accept json
// @Param stylist body models.Stylist true "Stylist"
// @Success 200 {object} dtos.Response
// @Failure 400,500 {object} dtos.Response
// @Router /stylist [post]
func CreateStylist(c *gin.Context) {
	var stylist models.Stylist
	token, _ := c.Cookie("token")
	verified, claims := utils.VerifyToken(token)
	if !verified {
		c.JSON(http.StatusUnauthorized, dtos.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    "session expired",
			Data:       nil,
		})
		return
	}

	if err := c.ShouldBindJSON(&stylist); err != nil {
		c.JSON(http.StatusBadRequest, dtos.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "required fields are missing",
			Data:       nil,
		})
		return
	}
	if claims.Id != stylist.UserId {
		c.JSON(http.StatusUnauthorized, dtos.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    "unauthorized",
			Data:       nil,
		})
		return
	}
	mongoUserId, _ := primitive.ObjectIDFromHex(stylist.UserId)
	err := userCollection.FindOne(
		context.TODO(),
		bson.D{
			{Key: "_id", Value: mongoUserId},
			{Key: "role", Value: "stylist"},
		}).Err()
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusBadRequest, dtos.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "user is not a stylist",
			Data:       nil,
		})
		return
	}
	if err != nil {
		utils.ErrorLogger(err)
		c.JSON(http.StatusInternalServerError, dtos.Response{
			StatusCode: http.StatusInternalServerError,
			Message:    "internal server error",
			Data:       nil,
		})
		return
	}
	err = stylistCollection.FindOne(context.TODO(), bson.D{{Key: "user_id", Value: stylist.UserId}}).Err()

	if err == mongo.ErrNoDocuments {
		stylist.CreatedAt = time.Now().Unix()
		stylist.UpdatedAt = time.Now().Unix()
		res, err := stylistCollection.InsertOne(context.TODO(), stylist)
		if err != nil {
			utils.ErrorLogger(err)
			c.JSON(http.StatusInternalServerError, dtos.Response{
				StatusCode: http.StatusInternalServerError,
				Message:    "internal server error",
				Data:       nil,
			})
		}
		var services []interface{}
		for _, val := range stylist.Services {
			services = append(services, bson.D{
				{Key: "name", Value: val.Name},
				{Key: "price", Value: val.Price},
				{Key: "currency_symbol", Value: val.CurrencySymbol},
				{Key: "currency_name", Value: val.CurrencyName},
				{Key: "stylist_id", Value: res.InsertedID},
			})
		}
		opts := options.InsertMany().SetOrdered(false)
		_, err = servicesCollection.InsertMany(context.TODO(), services, opts)
		if err != nil {
			utils.ErrorLogger(err)
			c.JSON(http.StatusInternalServerError, dtos.Response{
				StatusCode: http.StatusInternalServerError,
				Message:    "internal server error",
				Data:       nil,
			})
			return
		}
		c.JSON(http.StatusCreated, dtos.Response{
			StatusCode: http.StatusCreated,
			Message:    "stylist created successfully",
			Data:       gin.H{"stylist_id": res.InsertedID},
		})
		return
	}
	if err != nil {
		utils.ErrorLogger(err)
		c.JSON(http.StatusInternalServerError, dtos.Response{
			StatusCode: http.StatusInternalServerError,
			Message:    "internal server error",
			Data:       nil,
		})
		return
	}
	c.JSON(http.StatusBadRequest, dtos.Response{
		StatusCode: http.StatusBadRequest,
		Message:    "stylist already exists",
		Data:       nil,
	})
}

// ReviewStylist ... Reviews stylist
// @Summary  Reviews stylist
// @Description  Reviews stylist
// @Tags Stylist
// @Accept json
// @Param user body models.Review true "Stylist"
// @Success 200 {object} dtos.Response
// @Failure 400,500 {object} dtos.Response
// @Router /stylist/review [post]
func ReviewStylist(c *gin.Context) {
	var review models.Review
	token, _ := c.Cookie("token")
	verified, claims := utils.VerifyToken(token)
	if !verified {
		c.JSON(http.StatusUnauthorized, dtos.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    "session expired",
			Data:       nil,
		})
		return
	}
	if err := c.ShouldBindJSON(&review); err != nil {
		c.JSON(http.StatusBadRequest, dtos.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "required fields are missing",
			Data:       nil,
		})
		return
	}
	if claims.Id != review.UserId {
		c.JSON(http.StatusUnauthorized, dtos.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    "unauthorized",
			Data:       nil,
		})
		return
	}
	var result models.Stylist
	mongoStylistId, err := primitive.ObjectIDFromHex(review.StylistId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dtos.Response{
			StatusCode: http.StatusInternalServerError,
			Message:    "internal server error",
			Data:       nil,
		})
		return
	}

	err = stylistCollection.FindOne(context.TODO(), bson.D{{Key: "_id", Value: mongoStylistId}}).Decode(&result)
	if err != nil {
		c.JSON(http.StatusNotFound, dtos.Response{
			StatusCode: http.StatusNotFound,
			Message:    "invalid stylist",
			Data:       nil,
		})
		return
	}

	if result.UserId == review.UserId {
		c.JSON(http.StatusBadRequest, dtos.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "you can't create a review for yourself",
			Data:       nil,
		})
		return
	}

	err = reviewCollection.FindOne(context.TODO(), bson.D{{Key: "user_id", Value: review.UserId}}).Err()
	if err == mongo.ErrNoDocuments {
		review.CreatedAt = time.Now().Unix()
		res, err := reviewCollection.InsertOne(context.TODO(), review)
		if err != nil {
			utils.ErrorLogger(err)
			c.JSON(http.StatusInternalServerError, dtos.Response{
				StatusCode: http.StatusInternalServerError,
				Message:    "internal server error",
				Data:       nil,
			})
		}
		c.JSON(http.StatusCreated, dtos.Response{
			StatusCode: http.StatusCreated,
			Message:    "review created successfully",
			Data:       gin.H{"stylist_id": res.InsertedID},
		})
	}
	if err != nil {
		utils.ErrorLogger(err)
		c.JSON(http.StatusInternalServerError, dtos.Response{
			StatusCode: http.StatusInternalServerError,
			Message:    "internal server error",
			Data:       nil,
		})
		return
	}
	c.JSON(http.StatusBadRequest, dtos.Response{
		StatusCode: http.StatusBadRequest,
		Message:    "user already made a review",
		Data:       nil,
	})
}

// ReviewStylist ... Reviews stylist
// @Summary  Reviews stylist
// @Description  Reviews stylist
// @Tags Stylist
// @Accept json
// @Param images formData file true "Stylist Images"
// @Success 200 {object} dtos.Response
// @Failure 400,500 {object} dtos.Response
// @Router /stylist/images [post]
func StylistImageUpload(c *gin.Context) {
	token, _ := c.Cookie("token")
	verified, claims := utils.VerifyToken(token)
	if !verified {
		c.JSON(http.StatusUnauthorized, dtos.Response{
			StatusCode: http.StatusUnauthorized,
			Message:    "session expired",
			Data:       nil,
		})
		return
	}
	userMongoId, _ := primitive.ObjectIDFromHex(claims.Id)
	err := userCollection.FindOne(
		context.TODO(),
		bson.D{
			{Key: "_id", Value: userMongoId},
			{Key: "role", Value: "stylist"},
		}).Err()
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusBadRequest, dtos.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "user is not a stylist",
			Data:       nil,
		})
		return
	}
	var stylist models.Stylist
	err = stylistCollection.FindOne(context.TODO(), bson.D{{Key: "user_id", Value: claims.Id}}).Decode(&stylist)
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusBadRequest, dtos.Response{
			StatusCode: http.StatusBadRequest,
			Message:    "please update you details before uploading image",
			Data:       nil,
		})
		return
	}

	form, _ := c.MultipartForm()
	files := form.File["images"]
	var imageList []interface{}
	var imageSList []models.Image
	for _, file := range files {
		openedFile, err := file.Open()

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				dtos.Response{
					StatusCode: http.StatusInternalServerError,
					Message:    "select a file to upload",
					Data:       nil,
				})
			return
		}
		defer openedFile.Close()

		url, err := services.NewMediaUpload().FileUpload(models.File{File: openedFile})
		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				dtos.Response{
					StatusCode: http.StatusInternalServerError,
					Message:    "error uploading file",
					Data:       nil,
				})
			return
		}
		createdTime := time.Now().Unix()
		imageList = append(imageList, bson.D{
			{Key: "type", Value: "stylist"},
			{Key: "url", Value: url},
			{Key: "stylist_id", Value: stylist.Id},
			{Key: "created_at", Value: createdTime},
		})
		imageSList = append(imageSList, models.Image{
			Type:      "stylist",
			Url:       url,
			CreatedAt: createdTime,
		})
	}
	_, err = imagesCollection.InsertMany(context.TODO(), imageList)
	if err != nil {
		utils.ErrorLogger(err)
		c.JSON(http.StatusInternalServerError, dtos.Response{
			StatusCode: http.StatusInternalServerError,
			Message:    "internal server error",
			Data:       nil,
		})
		return
	}
	c.JSON(http.StatusCreated, dtos.Response{
		StatusCode: http.StatusCreated,
		Message:    "stylist images uploaded",
		Data:       gin.H{"images": imageSList},
	})
}

// func ServiceImageUpload(c *gin.Context) {

// }
// func ProductImageUpload(c *gin.Context) {

// }
