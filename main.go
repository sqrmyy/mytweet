package main

import (
	"log"
	"net/http"
	"strconv"

	"mytweet/crypto"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql" //直接的な記述が無いが、インポートしたいものに対しては"_"を頭につける決まり
	"github.com/jinzhu/gorm"
)

// Tweet モデル宣言
// モデルはDBのテーブル構造をGOの構造体で表したもの
type Tweet struct {
	gorm.Model
	Content string `form:"content" binding:"required"`
}

// User モデルの宣言
type User struct {
	gorm.Model
	Username string `form:"username" binding:"required" gorm:"unique;not null"`
	Password string `form:"password" binding:"required"`
}

func gormConnect() *gorm.DB {
	DBMS := "mysql"
	USER := "test"
	PASS := "12345678"
	DBNAME := "test"
	// MySQLだと文字コードの問題で"?parseTime=true"を末尾につける必要がある
	CONNECT := USER + ":" + PASS + "@/" + DBNAME + "?parseTime=true"
	db, err := gorm.Open(DBMS, CONNECT)

	if err != nil {
		panic(err.Error())
	}
	return db
}

// DBの初期化
func dbInit() {
	db := gormConnect()

	// コネクション解放解放
	defer db.Close()
	db.AutoMigrate(&Tweet{}) //構造体に基づいてテーブルを作成
	db.AutoMigrate(&User{})
}

// データインサート処理
func dbInsert(content string) {
	db := gormConnect()

	defer db.Close()
	// Insert処理
	db.Create(&Tweet{Content: content})
}

//DB更新
func dbUpdate(id int, tweetText string) {
	db := gormConnect()
	var tweet Tweet
	db.First(&tweet, id)
	tweet.Content = tweetText
	db.Save(&tweet)
	db.Close()
}

// 全件取得
func dbGetAll() []Tweet {
	db := gormConnect()

	defer db.Close()
	var tweets []Tweet
	// FindでDB名を指定して取得した後、orderで登録順に並び替え
	db.Order("created_at desc").Find(&tweets)
	return tweets
}

//DB一つ取得
func dbGetOne(id int) Tweet {
	db := gormConnect()
	var tweet Tweet
	db.First(&tweet, id)
	db.Close()
	return tweet
}

//DB削除
func dbDelete(id int) {
	db := gormConnect()
	var tweet Tweet
	db.First(&tweet, id)
	db.Delete(&tweet)
	db.Close()
}

// ユーザー登録処理
func createUser(username string, password string) []error {
	passwordEncrypt, _ := crypto.PasswordEncrypt(password)
	db := gormConnect()
	defer db.Close()
	// Insert処理
	if err := db.Create(&User{Username: username, Password: passwordEncrypt}).GetErrors(); err != nil {
		return err
	}
	return nil

}

// ユーザーを一件取得
func getUser(username string) User {
	db := gormConnect()
	var user User
	db.First(&user, "username = ?", username)
	db.Close()
	return user
}

func main() {
	router := gin.Default()
	router.LoadHTMLGlob("views/*.html")

	dbInit()

	//一覧
	router.GET("/", func(c *gin.Context) {
		tweets := dbGetAll()
		c.HTML(200, "index.html", gin.H{"tweets": tweets})
	})

	//登録
	router.POST("/new", func(c *gin.Context) {
		var form Tweet
		// ここがバリデーション部分
		if err := c.Bind(&form); err != nil {
			tweets := dbGetAll()
			c.HTML(http.StatusBadRequest, "index.html", gin.H{"tweets": tweets, "err": err})
			c.Abort()
		} else {
			content := c.PostForm("content")
			dbInsert(content)
			c.Redirect(302, "/")
		}
	})

	//投稿詳細
	router.GET("/detail/:id", func(c *gin.Context) {
		n := c.Param("id")
		id, err := strconv.Atoi(n)
		if err != nil {
			panic(err)
		}
		tweet := dbGetOne(id)
		c.HTML(200, "detail.html", gin.H{"tweet": tweet})
	})

	//更新
	router.POST("/update/:id", func(c *gin.Context) {
		n := c.Param("id")
		id, err := strconv.Atoi(n)
		if err != nil {
			panic("ERROR")
		}
		tweet := c.PostForm("tweet")
		dbUpdate(id, tweet)
		c.Redirect(302, "/")
	})

	//削除確認
	router.GET("/delete_check/:id", func(c *gin.Context) {
		n := c.Param("id")
		id, err := strconv.Atoi(n)
		if err != nil {
			panic("ERROR")
		}
		tweet := dbGetOne(id)
		c.HTML(200, "delete.html", gin.H{"tweet": tweet})
	})

	//削除
	router.POST("/delete/:id", func(c *gin.Context) {
		n := c.Param("id")
		id, err := strconv.Atoi(n)
		if err != nil {
			panic("ERROR")
		}
		dbDelete(id)
		c.Redirect(302, "/")

	})

	// ユーザー登録画面
	router.GET("/signup", func(c *gin.Context) {

		c.HTML(200, "signup.html", gin.H{})
	})

	// ユーザー登録
	router.POST("/signup", func(c *gin.Context) {
		var form User
		// バリデーション処理
		if err := c.Bind(&form); err != nil {
			c.HTML(http.StatusBadRequest, "signup.html", gin.H{"err": err})
			c.Abort()
		} else {
			username := c.PostForm("username")
			password := c.PostForm("password")
			// 登録ユーザーが重複していた場合にはじく処理
			if err := createUser(username, password); err != nil {
				c.HTML(http.StatusBadRequest, "signup.html", gin.H{"err": err})
			}
			c.Redirect(302, "/")
		}
	})

	// ユーザーログイン画面
	router.GET("/login", func(c *gin.Context) {

		c.HTML(200, "login.html", gin.H{})
	})

	// ユーザーログイン
	router.POST("/login", func(c *gin.Context) {

		// DBから取得したユーザーパスワード(Hash)
		dbPassword := getUser(c.PostForm("username")).Password
		log.Println(dbPassword)
		// フォームから取得したユーザーパスワード
		formPassword := c.PostForm("password")

		// ユーザーパスワードの比較
		if err := crypto.CompareHashAndPassword(dbPassword, formPassword); err != nil {
			log.Println("ログインできませんでした")
			c.HTML(http.StatusBadRequest, "login.html", gin.H{"err": err})
			c.Abort()
		} else {
			log.Println("ログインできました")
			c.Redirect(302, "/")
		}
	})

	router.Run()
}
