package client

import (
	"context"
	"errors"
	pb "github.com/piroyoung/lanterne/grpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
	"math"
	"strconv"
	"time"
)

type LanterneClient struct {
	conn   *grpc.ClientConn
	client pb.LanterneClient
}

func NewLanterneClient(hostname string, port int) (*LanterneClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	chConn := make(chan *grpc.ClientConn)
	chErr := make(chan error)

	go func() {
		conn, err := grpc.DialContext(ctx, hostname+":"+strconv.Itoa(port), grpc.WithInsecure())
		if err != nil {
			chErr <- err
			return
		}
		chConn <- conn
	}()
	select {
	case <-ctx.Done():
		return nil, errors.New("grpc connection timeout")

	case err := <-chErr:
		return nil, err

	case conn := <-chConn:
		return &LanterneClient{
			conn:   conn,
			client: pb.NewLanterneClient(conn),
		}, nil
	}
}

func (c *LanterneClient) Close() error {
	return c.conn.Close()
}

func (c *LanterneClient) DumpEdge(ctx context.Context, tail string, head string, weight float32) error {
	tailVertex, err := newVertex(tail, nil)
	if err != nil {
		return err
	}
	headVertex, err := newVertex(head, nil)
	if err != nil {
		return err
	}
	edge := &pb.Edge{
		Tail:   tailVertex,
		Head:   headVertex,
		Weight: weight,
	}
	response, err := c.client.DumpEdge(ctx, edge)
	if err != nil {
		return err
	}
	if response.Status != pb.Status_OK {
		return errors.New("dump edge error")
	}
	return nil
}

func (c *LanterneClient) DumpVertex(ctx context.Context, key string, value interface{}) error {
	vertex, err := newVertex(key, value)
	if err != nil {
		return err
	}
	response, err := c.client.DumpVertex(ctx, vertex)
	if err != nil {
		return err
	}
	if response.Status != pb.Status_OK {
		return errors.New("dump vertex error. status: " + pb.Status_OK.String())
	}
	return nil
}

func (c *LanterneClient) Illuminate(ctx context.Context, seed string, step uint32) (*pb.Graph, error) {
	request := &pb.IlluminateRequest{
		Seed:      &pb.Vertex{Key: seed},
		Step:      step,
		MinWeight: -math.MaxFloat32,
		MaxWeight: math.MaxFloat32,
	}
	response, err := c.client.Illuminate(ctx, request)
	if err != nil {
		return nil, err
	}
	if response.Status != pb.Status_OK {
		return nil, errors.New("illuminate error. status: " + response.Status.String())
	}
	return response.Graph, nil
}

func newVertex(key string, value interface{}) (*pb.Vertex, error) {
	vertex := &pb.Vertex{
		Key: key,
	}
	switch v := value.(type) {
	case float64:
		vertex.Value = &pb.Vertex_Float64{Float64: v}

	case float32:
		vertex.Value = &pb.Vertex_Float32{Float32: v}

	case int32:
		vertex.Value = &pb.Vertex_Int32{Int32: v}

	case int64:
		vertex.Value = &pb.Vertex_Int64{Int64: v}

	case uint32:
		vertex.Value = &pb.Vertex_Uint32{Uint32: v}

	case uint64:
		vertex.Value = &pb.Vertex_Uint64{Uint64: v}

	case bool:
		vertex.Value = &pb.Vertex_Bool{Bool: v}

	case string:
		vertex.Value = &pb.Vertex_String_{String_: v}

	case []byte:
		vertex.Value = &pb.Vertex_Bytes{Bytes: v}

	case time.Time:
		vertex.Value = &pb.Vertex_Timestamp{Timestamp: timestamppb.New(v)}

	case nil:
		vertex.Value = &pb.Vertex_Nil{Nil: true}

	default:
		return nil, errors.New("type mismatch")
	}
	return vertex, nil
}