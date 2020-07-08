package rpcx

import (
	"fmt"
	"strings"

	pb "github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
)

const (
	rpcxServerPkgPath   = "github.com/smallnest/rpcx/v5/server"
	rpcxClientPkgPath   = "github.com/smallnest/rpcx/v5/client"
	rpcxProtocolPkgPath = "github.com/smallnest/rpcx/v5/protocol"
)

func init() {
	generator.RegisterPlugin(new(rpcx))
}

type rpcx struct {
	gen *generator.Generator
}

//Name returns the name of this plugin
func (p *rpcx) Name() string {
	return "rpcx"
}

//Init initializes the plugin.
func (p *rpcx) Init(gen *generator.Generator) {
	p.gen = gen
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (p *rpcx) objectNamed(name string) generator.Object {
	p.gen.RecordTypeUse(name)
	return p.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (p *rpcx) typeName(str string) string {
	return p.gen.TypeName(p.objectNamed(str))
}

// GenerateImports generates the import declaration for this file.
func (p *rpcx) GenerateImports(file *generator.FileDescriptor) {
}

// P forwards to g.gen.P.
func (p *rpcx) P(args ...interface{}) { p.gen.P(args...) }

// Generate generates code for the services in the given file.
func (p *rpcx) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}
	_ = p.gen.AddImport(rpcxServerPkgPath)
	_ = p.gen.AddImport(rpcxClientPkgPath)
	_ = p.gen.AddImport(rpcxProtocolPkgPath)
	_ = p.gen.AddImport("context")

	// generate all services
	for i, service := range file.FileDescriptorProto.Service {
		p.generateService(file, service, i)
	}
}

// generateService generates all the code for the named service
func (p *rpcx) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	originServiceName := service.GetName()
	serviceName := upperFirstLatter(originServiceName)
	p.P("// This following code was generated by rpcx")
	p.P(fmt.Sprintf("// Gernerated from %s", file.GetName()))
	p.P()
	p.P("//================== server skeleton===================")
	p.P(fmt.Sprintf("type %s interface {", serviceName))
	for _, method := range service.Method {
		p.generateServerCode(service, method)
	}
	p.P(fmt.Sprintf(`}

		// RegisterFor%[1]s register the '%[1]s' service to a server.
		// And will check all methods of '%[1]s' which must be implemented.
		func RegisterFor%[1]s(s *server.Server, meta string) error {
			impl := new(%[1]sImpl)%[2]s
			return s.RegisterName("%[1]s", impl, meta)
		}
	`, serviceName, func() (methods string) {
		for _, method := range service.Method {
			methods += fmt.Sprintf("\n_ = impl.%s", upperFirstLatter(method.GetName()))
		}
		return
	}()))
	p.P()
	p.P()
	p.P("//================== client stub===================")
	p.P(fmt.Sprintf(`// %[1]s is a client wrapped XClient.
		type %[1]sClient struct {
			xclient client.XClient
			fx %[1]sMethods
		}

		type %[1]sMethods struct {%[2]s}

		// Make%[1]sClient wraps a XClient as %[1]sClient.
		// You can pass a shared XClient object created by anywhere.
		func Make%[1]sClient(xclient client.XClient) *%[1]sClient {
			return &%[1]sClient{
				xclient: xclient,
				fx: UserMethods {%[3]s
				},
			}
		}
	`, serviceName, func() (methods string) {
		for _, method := range service.Method {
			methods += fmt.Sprintf("\n%s string", upperFirstLatter(method.GetName()))
		}
		return
	}(), func() (methods string) {
		for _, method := range service.Method {
			methods += fmt.Sprintf("\n%s: \"%s\",", upperFirstLatter(method.GetName()), method.GetName())
		}
		return
	}()))
	for _, method := range service.Method {
		p.generateClientCode(service, method)
	}
}

func (p *rpcx) generateServerCode(service *pb.ServiceDescriptorProto, method *pb.MethodDescriptorProto) {
	methodName := upperFirstLatter(method.GetName())
	inType := p.typeName(method.GetInputType())
	outType := p.typeName(method.GetOutputType())
	p.P(fmt.Sprintf(`// %s is server rpc method as defined
		%s(ctx context.Context, args *%s, reply *%s) error
	`, methodName, methodName, inType, outType))
}

func (p *rpcx) generateClientCode(service *pb.ServiceDescriptorProto, method *pb.MethodDescriptorProto) {
	methodName := upperFirstLatter(method.GetName())
	serviceName := upperFirstLatter(service.GetName())
	inType := p.typeName(method.GetInputType())
	outType := p.typeName(method.GetOutputType())
	p.P(fmt.Sprintf(`// %[2]s is client rpc method as defined
		func (c *%[1]sClient) %[2]s(ctx context.Context, args *%[3]s) (reply *%[4]s, err error) {
			reply = &%[4]s{}
			err = c.xclient.Call(ctx, "%[5]s", args, reply)
			return reply, err
		}
		
		// %[2]sGo is client rpc method as defined
		func (c *%[1]sClient) %[2]sGo(ctx context.Context, args *%[3]s, done chan *client.Call) (call *client.Call, err error) {
			return c.xclient.Go(ctx, "%[5]s", args, &%[4]s{}, done)
		}
	`, serviceName, methodName, inType, outType, method.GetName()))
}

// upperFirstLatter make the fisrt charater of given string  upper class
func upperFirstLatter(s string) string {
	if len(s) == 0 {
		return ""
	}
	if len(s) == 1 {
		return strings.ToUpper(string(s[0]))
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}
