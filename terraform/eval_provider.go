package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
)

func buildProviderConfig(ctx EvalContext, addr addrs.ProviderConfig, body hcl.Body) hcl.Body {
	// If we have an Input configuration set, then merge that in
	if input := ctx.ProviderInput(addr); input != nil {
		// "input" is a map of the subset of config values that were known
		// during the input walk, set by EvalInputProvider. Note that
		// in particular it does *not* include attributes that had
		// computed values at input time; those appear *only* in
		// "cfg" here.

		inputBody := configs.SynthBody("<input prompt>", input)
		body = configs.MergeBodies(body, inputBody)
	}

	return body
}

// EvalConfigProvider is an EvalNode implementation that configures
// a provider that is already initialized and retrieved.
type EvalConfigProvider struct {
	Addr     addrs.ProviderConfig
	Provider *ResourceProvider
	Config   *configs.Provider
}

func (n *EvalConfigProvider) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics
	provider := *n.Provider
	config := *n.Config

	schema, err := provider.GetSchema(&ProviderSchemaRequest{})
	if err != nil {
		diags = diags.Append(err)
		return nil, diags.NonFatalErr()
	}

	configSchema := schema.Provider
	configBody := buildProviderConfig(ctx, n.Addr, config.Config)
	configVal, configBody, evalDiags := ctx.EvaluateBlock(configBody, configSchema, nil)
	diags = diags.Append(evalDiags)
	if evalDiags.HasErrors() {
		return nil, diags.NonFatalErr()
	}

	configDiags := ctx.ConfigureProvider(n.Addr, configVal)
	configDiags = configDiags.InConfigBody(configBody)

	return nil, configDiags.ErrWithWarnings()
}

// EvalInitProvider is an EvalNode implementation that initializes a provider
// and returns nothing. The provider can be retrieved again with the
// EvalGetProvider node.
type EvalInitProvider struct {
	TypeName string
	Addr     addrs.ProviderConfig
}

func (n *EvalInitProvider) Eval(ctx EvalContext) (interface{}, error) {
	return ctx.InitProvider(n.TypeName, n.Addr)
}

// EvalCloseProvider is an EvalNode implementation that closes provider
// connections that aren't needed anymore.
type EvalCloseProvider struct {
	Addr addrs.ProviderConfig
}

func (n *EvalCloseProvider) Eval(ctx EvalContext) (interface{}, error) {
	ctx.CloseProvider(n.Addr)
	return nil, nil
}

// EvalGetProvider is an EvalNode implementation that retrieves an already
// initialized provider instance for the given name.
type EvalGetProvider struct {
	Addr   addrs.ProviderConfig
	Output *ResourceProvider
}

func (n *EvalGetProvider) Eval(ctx EvalContext) (interface{}, error) {
	result := ctx.Provider(n.Addr)
	if result == nil {
		return nil, fmt.Errorf("provider %s not initialized", n.Addr)
	}

	if n.Output != nil {
		*n.Output = result
	}

	return nil, nil
}

// EvalInputProvider is an EvalNode implementation that asks for input
// for the given provider configurations.
type EvalInputProvider struct {
	Addr     addrs.ProviderConfig
	Provider *ResourceProvider
	Config   *configs.Provider
}

func (n *EvalInputProvider) Eval(ctx EvalContext) (interface{}, error) {
	// This is currently disabled. It used to interact with a provider method
	// called Input, allowing the provider to capture input interactively
	// itself, but once re-implemented we'll have this instead use the
	// provider's configuration schema to automatically infer what we need
	// to prompt for.
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.SimpleWarning(fmt.Sprintf("%s: provider input is temporarily disabled", n.Addr)))
	return nil, diags.ErrWithWarnings()
}
