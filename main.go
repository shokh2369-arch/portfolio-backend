package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"net/http"
	"net/url"
	"strconv"

	_ "example.com/portfolio/docs"

	"example.com/portfolio/admin"
	"example.com/portfolio/content"
	"example.com/portfolio/db"
	"example.com/portfolio/info"
	"example.com/portfolio/middlewares"
	"example.com/portfolio/utils"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	ginSwagger "github.com/swaggo/gin-swagger"

	// gin-swagger middleware
	// swagger embed files
	swaggerFiles "github.com/swaggo/files"
)

var (
	botToken string
	adminID  string
)

func init() {
	_ = godotenv.Load()
	botToken = os.Getenv("botToken")
	adminID = os.Getenv("adminID")
	if botToken == "" || adminID == "" {
		log.Fatal("botToken or adminID not set in environment")
	}
}

// @title Portfolio API
// @version 1.0
// @description This is the portfolio back-end
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host portfolioshokh.onrender.com
// @BasePath /
// @schemes https

func main() {
	db.Initdb()
	r := gin.Default()

	auth := r.Group("/")
	auth.Use(middlewares.Authenticate)
	{
		auth.POST("/post", publishBlog)
		auth.PUT("/update/:id", editBlog)
		auth.DELETE("/delete/:id", deleteBlog)
	}
	fmt.Println("SHOW_SIGNUP =", os.Getenv("SHOW_SIGNUP"))
	r.GET("/blog/:id", getSingle)
	r.GET("/portfolio", hello)
	r.GET("/blogs/:page", blogs)
	r.POST("/request", request)
	showSignup := os.Getenv("SHOW_SIGNUP")
	if showSignup == "true" {
		r.POST("/signup", register)
		log.Println("Signup route enabled ✅")
	} else {
		log.Println("Signup route hidden 🚫")
	}
	r.POST("/login", login)

	// ✅ Swagger route must be BEFORE Run
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // fallback for local testing
	}
	r.Run("0.0.0.0:" + port)
}

// hello godoc
// @Summary      Hello endpoint
// @Description  Returns Hello World message
// @Tags         general
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /portfolio [get]
func hello(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Hello world"})
}

// request handles portfolio requests and sends a Telegram notification.
//
// @Summary      Submit a portfolio request
// @Description  Accepts user portfolio request data, saves it, and notifies admin via Telegram
// @Tags         requests
// @Accept       json
// @Produce      json
// @Param        request  body      info.About  true  "Portfolio request data"
// @Success      200      {object}  map[string]interface{}  "Request successfully submitted"
// @Failure      400      {object}  map[string]string        "Invalid input"
// @Failure      429      {object}  map[string]string        "Daily request limit reached"
// @Failure      500      {object}  map[string]string        "Server or database error"
// @Router       /request [post]
func request(c *gin.Context) {
	var i info.About

	if err := c.ShouldBindJSON(&i); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ip := c.ClientIP()

	ok, err := info.CanRequest(ip)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "Daily request limit reached (2 per day)",
		})
		return
	}

	if err := i.Save(ip); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	msg := fmt.Sprintf(
		"📩 New portfolio request\n\n👤 %s %s\n📞 %s\n💬 Telegram: %s\n📝 %s\n🌐 IP: %s",
		i.Name, i.Lastname, i.Phone, i.Telegram, i.Description, ip,
	)

	if err := sendTelegramMessage(botToken, adminID, msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send Telegram notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Your request was sent successfully",
		"request": i,
	})
}

func sendTelegramMessage(token, chatID, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	resp, err := http.PostForm(apiURL, url.Values{
		"chat_id": {chatID},
		"text":    {message},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("Telegram response:", string(body))

	return nil
}

// @Summary      Publish new blog
// @Description  Creates a new content entry (blog or project) in the database
// @Security     TokenAuth
// @Tags         content
// @Accept       json
// @Produce      json
// @Param        content  body      content.Content  true  "Blog data"
// @Success      201      {object}  content.Content  "Blog created successfully"
// @Failure      400      {object}  map[string]string  "Invalid request"
// @Failure      500      {object}  map[string]string  "Database or server error"
// @Router       /post [post]
func publishBlog(c *gin.Context) {
	var k content.Content
	if err := c.ShouldBindJSON(&k); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := k.Add(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not publish the blog", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, k)
}

// blog edit godoc
// @Summary for deleting the blog contents
// @Description deletes blog
// @Security TokenAuth
// @Tags content
// @Accept json
// @Produce json
// @Param id path int true "ID number to fetch"
// @Param content body content.Content true "Blog data"
// @Success 200 {object} map[string]string  "Blog updated successfully"
// @Failure 400 {object} map[string]string "Invalid blog ID"
// @Failure 400 {object} map[string]string  "Could not find blog with this ID"
// @Failure 500 {object} map[string]string "Invalid request body"
// @Router /update/{id} [put]
func editBlog(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blog ID"})
		return
	}

	cnt, err := content.GetById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Could not find blog with this ID"})
		return
	}

	if err := c.ShouldBindJSON(&cnt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	cnt.ID = id

	if err := cnt.Update(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update blog"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Blog updated successfully",
		"content": cnt,
	})
}

// blog delete godoc
// @Summary for deleting the blog
// @Description deletes blog
// @Security TokenAuth
// @Tags content
// @Accept json
// @Produce json
// @Param id path int true "ID number to fetch"
// @Success 200 {object} map[string]string  "Blog deleted successfully"
// @Failure 400 {object} map[string]string "Invalid blog ID"
// @Failure 400 {object} map[string]string  "Could not find blog with this ID"
// @Failure 500 {object} map[string]string "Failed to delete blog"
// @Router /delete/{id} [delete]
func deleteBlog(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blog ID"})
		return
	}

	cnt, err := content.GetById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Could not find blog with this ID"})
		return
	}

	cnt.ID = id
	if err := cnt.Delete(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete blog"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Blog deleted successfully"})
}

// blogs godoc
// @Summary      Get blogs
// @Description  Returns paginated blogs with optional filters for language, category, and title.
// @Tags         Content
// @Param        page        path      int     true   "Page number"
// @Param        language    query     string  false  "Language filter (default: en)"  Enums(en, ru, uz)
// @Param        category    query     string  false  "Category filter"                Enums(blog, project)
// @Param        title       query     string  false  "Search by blog title"
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Blogs retrieved successfully"
// @Failure      400  {object}  map[string]string       "Invalid parameters"
// @Failure      404  {object}  map[string]string       "No blogs found"
// @Failure      500  {object}  map[string]string       "Internal server error"
// @Router       /blogs/{page} [get]
func blogs(c *gin.Context) {
	page, err := strconv.ParseInt(c.Param("page"), 10, 64)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page number"})
		return
	}

	language := c.DefaultQuery("language", "en")
	if language != "en" && language != "ru" && language != "uz" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid language"})
		return
	}

	category := c.Query("category")
	if category != "blog" && category != "project" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category"})
		return
	}
	title := c.Query("title")

	featured := c.Query("featured")
	if featured != "" && featured != "true" && featured != "false" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid featured value"})
		return
	}

	contents, err := content.GetContents(title, int(page), language, category, featured)
	if err != nil {
		if err.Error() == "no contents found" {
			c.JSON(http.StatusNotFound, gin.H{"message": "No blogs found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to fetch blogs",
			"detail": err.Error(),
		})
		return
	}

	if title == "" {
		c.JSON(http.StatusOK, gin.H{
			"message":  "All blogs fetched successfully",
			"contents": contents,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message":  "We found these blogs",
			"contents": contents,
		})
	}
}

// register godoc
// @Summary Sign up admin
// @Description Register sign up in order to login
// @Tags admin
// @Accept json
// @Produce json
// @Param admin body admin.Address true "Admin sign up"
// @Success 201 {object} map[string]string  "Signed up successfully"
// @Failure 400 {object} map[string]string "The sign up must contain username, email, password"
// @Failure 500 {object} map[string]string  "Could not sign up. Try again later"
// @Router /signup [post]
func register(c *gin.Context) {
	var a admin.Address
	err := c.ShouldBindJSON(&a)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "The sign up must contain username, email, password"})
		return
	}
	err = a.SignUp()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not sign up. Try again later"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Signed up successfully"})
}

// login godoc
// @Summary login
// @Description Admin's login page
// @Tags admin
// @Accept json
// @Produce json
// @Param admin body admin.Login true "Admin login"
// @Success 200 {object} map[string]string  "Logged in successfully"
// @Failure 400 {object} map[string]string "Invalid login"
// @Failure 400 {object} map[string]string  "Invalid input"
// @Failure 500 {object} map[string]string "Could not generate token"
// @Router /login [post]
func login(c *gin.Context) {
	var l admin.Login
	err := c.ShouldBindJSON(&l)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid login"})
		return
	}

	s := admin.Address{
		Username: l.Login,
		Email:    l.Login,
		Password: l.Password,
	}
	err = s.Login()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	token, err := utils.GenerateToken(s.Username, s.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged in successfully", "token": token})

}

// getSingle godoc
// @Summary Get single content by ID
// @Description Returns one content item (blog or project) by its ID
// @Tags content
// @Produce json
// @Param id path int true "Blog ID"
// @Success 200 {object} content.Content
// @Failure 400 {object} map[string]string "Invalid blog ID"
// @Failure 404 {object} map[string]string "Blog not found"
// @Router /blog/{id} [get]
func getSingle(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blog ID"})
		return
	}

	content, err := content.GetById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Blog not found"})
		return
	}

	c.JSON(http.StatusOK, content)
}
