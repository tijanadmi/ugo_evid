package api

import (
	"fmt"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	swaggerFiles "github.com/swaggo/files"     // swagger embed files
	ginSwagger "github.com/swaggo/gin-swagger" // gin-swagger middleware
	db "github.com/tijanadmi/ugo_evid/repository"
	"github.com/tijanadmi/ugo_evid/token"
	"github.com/tijanadmi/ugo_evid/util"
)

// Server serves HTTP requests for our banking service.
type Server struct {
	config           util.Config
	store            db.Store
	tokenMaker       token.Maker
	router           *gin.Engine
	authenticateLDAP func(util.LDAPConfig, string, string) error
}

// NewServer creates a new HTTP server and set up routing.
func NewServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:           config,
		authenticateLDAP: util.AuthenticateLDAP,
		store:            store,
		tokenMaker:       tokenMaker,
	}
	/*if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}*/

	server.setupRouter()
	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.Default()

	// Configure CORS
	router.Use(cors.New(cors.Config{
		//AllowOrigins:     []string{"http://localhost:3000"},
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	router.POST("/users/login", server.loginUser)
	authRoutes := router.Group("/").Use(authMiddleware(server.tokenMaker))

	authRoutes.GET("/sapugovori", server.GetSapUgovoriPaged)
	authRoutes.GET("/ugo_evid/otvoreni", server.ListOtvoreniUgovori)
	authRoutes.GET("/ugo_evid/zatvoreni", server.ListZatvoreniUgovori)
	authRoutes.GET("/ugo_evid/:id/detalji", server.GetUgoEvidProsireni)

	// CRUD
	authRoutes.GET("/ugo_org/:id", server.GetUgoOrg)       // Get po id (int)
	authRoutes.POST("/ugo_org", server.InsertUgoOrg)       // Insert
	authRoutes.PUT("/ugo_org", server.UpdateUgoOrg)        // Update
	authRoutes.DELETE("/ugo_org/:id", server.DeleteUgoOrg) // Delete po id

	// List sa filterom i paginacijom
	authRoutes.GET("/ugo_org", server.ListUgoOrg)

	// CRUD
	authRoutes.GET("/ugo_dob_lica_rola/:id", server.GetUgoDobLicaRolaById)   // Get po id (int)
	authRoutes.POST("/ugo_dob_lica_rola", server.InsertUgoDobLicaRola)       // Insert
	authRoutes.PUT("/ugo_dob_lica_rola/:id", server.UpdateUgoDobLicaRola)    // Update
	authRoutes.DELETE("/ugo_dob_lica_rola/:id", server.DeleteUgoDobLicaRola) // Delete po id

	// List sa filterom i paginacijom
	authRoutes.GET("/ugo_dob_lica_rola", server.ListUgoDobLicaRola)

	// CRUD
	authRoutes.GET("/ugo_dob_lica/:id", server.GetUgoDobLice)       // Get po id (int)
	authRoutes.POST("/ugo_dob_lica", server.InsertUgoDobLice)       // Insert
	authRoutes.PUT("/ugo_dob_lica/:id", server.UpdateUgoDobLice)    // Update
	authRoutes.DELETE("/ugo_dob_lica/:id", server.DeleteUgoDobLice) // Delete po id

	// List sa filterom i paginacijom
	authRoutes.GET("/ugo_dob_lica", server.ListUgoDobLice)

	// List sa filterom i paginacijom

	// CRUD
	authRoutes.GET("/ugo_evid/:id", server.GetUgoEvid)       // Get po id (int)
	authRoutes.POST("/ugo_evid", server.InsertUgoEvid)       // Insert
	authRoutes.PUT("/ugo_evid/:id", server.UpdateUgoEvid)    // Update
	authRoutes.DELETE("/ugo_evid/:id", server.DeleteUgoEvid) // Delete po id

	// List sa filterom i paginacijom
	authRoutes.GET("/ugo_evid", server.ListUgoEvid)

	router.POST("/tokens/renew_access", server.renewAccessToken)

	// router.GET("/halls/:id", server.getHallById)
	// router.GET("/halls", server.listHalls)
	// router.POST("/halls", server.InsertHall)
	// router.PUT("/halls/:id", server.UpdateHall)
	// router.DELETE("/halls/:id", server.DeleteHall)
	// router.GET("/searchhalls/:name", server.searchHall)

	authRoutes.GET("/users", server.getUserByUsername)

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	server.router = router
}

// Start runs the HTTP server on a specific address.
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}
