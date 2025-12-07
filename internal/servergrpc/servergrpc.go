package servergrpc

import (
	"context"
	"errors"
	"fmt"
	config "metralert/config/server"
	"metralert/internal/metrics"
	pb "metralert/internal/proto"
	"metralert/internal/storage"
	"net"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type ServerGRPC struct {
	pb.UnimplementedMetricsServer

	address string
	storage storage.StorageInterface
	logger  *zap.SugaredLogger
}

func New(cfg config.Config) *ServerGRPC {
	server := &ServerGRPC{
		address: cfg.ServerAddress,
		storage: cfg.Storage,
		logger:  cfg.Logger,
	}

	return server
}

func (server *ServerGRPC) Start() error {
	server.logger.Infow(
		"Starting GRPC server",
		"url", server.address,
	)

	addressSlice := strings.Split(server.address, ":")
	if len(addressSlice) != 2 {
		return errors.New("invalid server address")
	}

	// hostname := addressSlice[0]
	port := addressSlice[1]

	listen, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return fmt.Errorf("unable to listen server: %s", err)
	}

	s := grpc.NewServer()

	pb.RegisterMetricsServer(s, server)

	server.logger.Infoln("GRPC server started")

	if err = s.Serve(listen); err != nil {
		server.logger.Errorln("error occured during server running: ", err)
		return err
	}
	return nil
}

func (server *ServerGRPC) UpdateMetrics(ctx context.Context, in *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {

	metricsSlice, err := ConvertMetricReqToMetric(in)
	if err != nil {
		return nil, err
	}
	_, err = server.storage.UpdateBatchMetrics(ctx, metricsSlice)
	if err != nil {
		return nil, err
	}

	response := pb.UpdateMetricsResponse_builder{}

	return response.Build(), nil

}

func ConvertMetricReqToMetric(req *pb.UpdateMetricsRequest) ([]metrics.Metrics, error) {
	metricsSlice := make([]metrics.Metrics, 0, len(req.GetMetrics()))

	for _, m := range req.GetMetrics() {
		delta := m.GetDelta()
		value := m.GetValue()
		mp := metrics.Metrics{
			ID:    m.GetId(),
			MType: strings.ToLower(m.GetType().String()),
			Delta: &delta,
			Value: &value,
		}

		metricsSlice = append(metricsSlice, mp)
	}
	if len(metricsSlice) == 0 {
		return nil, errors.New("got empty metrics slice while converting")
	}
	return metricsSlice, nil
}
