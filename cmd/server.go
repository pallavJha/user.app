package cmd

import (
	"fmt"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/rs/zerolog/log"
	"github.com/soheilhy/cmux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"net"
	"user.app/message"
	"user.app/pkg/api"
	"user.app/pkg/auth"
	"user.app/pkg/conn"
)

var applicationCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the userapp RPC service",
	Run: func(cmd *cobra.Command, args []string) {
		boil.DebugMode = true
		var err error
		conn.Instance, err = conn.InitDBConnection()
		if err != nil {
			log.Error().Err(err).Msg("cannot initiate the connection with the database")
		}
		authenticator, err := auth.NewJWTAuth()
		if err != nil {
			log.Fatal().Err(err).Msg("jwt authenticator initialization failed")
		}
		auth.Authenticator = authenticator

		serviceAddr := fmt.Sprintf("%s:%d", viper.GetString("host"), viper.GetInt("port"))
		listener, err := net.Listen("tcp", serviceAddr)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to start cmux listener")
		}

		m := cmux.New(listener)
		http2Listener := m.Match(cmux.HTTP2())
		serverObj := &api.Server{}

		g := new(errgroup.Group)
		g.Go(func() error {
			gRPCServer := grpc.NewServer(
				grpc.UnaryInterceptor(
					grpc_middleware.ChainUnaryServer(
						auth.UnaryServerInterceptor(),
					),
				),
			)

			message.RegisterUserAppServer(gRPCServer, serverObj)
			return gRPCServer.Serve(http2Listener)
		})
		g.Go(func() error { return m.Serve() })
		log.Info().Msgf("application server listening on %s", serviceAddr)
		if err := g.Wait(); err != nil {
			log.Fatal().Err(err).Msg("err-group first non-nil error")
		}
	},
}

func init() {
	applicationCmd.PersistentFlags().StringP("host", "n", "localhost", "Service Host")
	if err := viper.BindPFlag("host", applicationCmd.PersistentFlags().Lookup("host")); err != nil {
		log.Fatal().Err(err).Msg("viper binding failed for host")
	}

	applicationCmd.PersistentFlags().StringP("port", "p", "9688", "Port")
	if err := viper.BindPFlag("port", applicationCmd.PersistentFlags().Lookup("port")); err != nil {
		log.Fatal().Err(err).Msg("viper binding failed for service.port")
	}

	RootCmd.AddCommand(applicationCmd)
}
