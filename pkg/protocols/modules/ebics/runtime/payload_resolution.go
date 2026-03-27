package runtime

import (
	"fmt"
	"maps"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	profileRequired  = "profile-required"
	profilePreferred = "profile-preferred"
)

type PayloadRequestInput struct {
	ProfileName string
	RuleName    string
	Subscriber  PayloadSubscriberRef
	File        *PayloadFileRef
	Target      *PayloadTargetRef
	Service     *PayloadServiceRef
	Metadata    map[string]any
}

type PayloadSubscriberRef struct {
	HostID    string
	PartnerID string
	UserID    string
}

type PayloadFileRef struct {
	Path       string
	OutputName string
}

type PayloadTargetRef struct {
	Directory string
}

type PayloadServiceRef struct {
	OrderType     string
	ServiceName   string
	ServiceOption string
	Scope         string
	MsgName       string
	ContainerType string
}

type ResolvedPayloadRequest struct {
	OrderType         string
	ResolutionMode    string
	Profile           *model.EbicsPayloadProfile
	ProfileName       string
	RuleName          string
	Subscriber        PayloadSubscriberRef
	ResolvedFile      *PayloadFileRef
	ResolvedTarget    *PayloadTargetRef
	ResolvedService   PayloadServiceRef
	ResolvedMetadata  map[string]any
	DeclaredAmount    string
	DeclaredCurrency  string
	ContractViewID    int64
	ContractItemIDs   []int64
	StandardCatalogID int64
	StandardEntryIDs  []int64
}

type PayloadProfileResolver interface {
	GetPayloadProfile(owner, name string) (*model.EbicsPayloadProfile, error)
}

func ResolvePayloadRequest(
	input *PayloadRequestInput,
	profilePolicy string,
	defaults map[string]any,
	resolver PayloadProfileResolver,
) (*ResolvedPayloadRequest, error) {
	if input == nil {
		return nil, database.NewValidationError("the EBICS payload request is missing")
	}

	profilePolicy = strings.TrimSpace(profilePolicy)
	if profilePolicy == "" {
		profilePolicy = profilePreferred
	}

	var profile *model.EbicsPayloadProfile
	if input.ProfileName != "" {
		if resolver == nil {
			return nil, database.NewValidationError("payload profile resolution requires a profile resolver")
		}

		var err error
		profile, err = resolver.GetPayloadProfile(conf.GlobalConfig.GatewayName, input.ProfileName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve EBICS payload profile %q: %w", input.ProfileName, err)
		}

		if !profile.IsEnabled {
			return nil, database.NewValidationErrorf(
				"the EBICS payload profile %q is disabled", input.ProfileName)
		}
	} else if profilePolicy == profileRequired {
		return nil, database.NewValidationError("an EBICS payload profile is required by policy")
	}

	service, err := resolvePayloadService(input, profile, defaults)
	if err != nil {
		return nil, err
	}

	if validateErr := validatePayloadOrderType(service.OrderType); validateErr != nil {
		return nil, validateErr
	}

	resolved := &ResolvedPayloadRequest{
		OrderType:        service.OrderType,
		ResolutionMode:   resolveResolutionMode(profile),
		Profile:          profile,
		ProfileName:      strings.TrimSpace(input.ProfileName),
		RuleName:         resolveRuleName(input, profile, defaults),
		Subscriber:       normalizePayloadSubscriber(input.Subscriber),
		ResolvedFile:     resolvePayloadFile(input),
		ResolvedTarget:   resolvePayloadTarget(input, profile),
		ResolvedService:  service,
		ResolvedMetadata: mergePayloadMetadata(input.Metadata, defaults),
	}

	resolved.DeclaredAmount, resolved.DeclaredCurrency = extractDeclaredAmount(resolved.ResolvedMetadata)
	if profile != nil && profile.RequiresDeclaredAmount && resolved.DeclaredAmount == "" {
		return nil, database.NewValidationError(
			"the selected EBICS payload profile requires a declared amount")
	}

	return resolved, nil
}

func resolvePayloadService(
	input *PayloadRequestInput,
	profile *model.EbicsPayloadProfile,
	defaults map[string]any,
) (PayloadServiceRef, error) {
	service := PayloadServiceRef{}
	if input.Service != nil {
		service = *input.Service
	}

	if profile != nil {
		if service.OrderType == "" {
			service.OrderType = profile.OrderType
		}

		if service.ServiceName == "" {
			service.ServiceName = profile.ServiceName
		}

		if service.ServiceOption == "" {
			service.ServiceOption = profile.ServiceOption
		}

		if service.Scope == "" {
			service.Scope = profile.Scope
		}

		if service.MsgName == "" {
			service.MsgName = profile.MsgName
		}

		if service.ContainerType == "" {
			service.ContainerType = profile.ContainerType
		}
	}

	setServiceStringDefault(&service.OrderType, defaults, "orderType")
	setServiceStringDefault(&service.ServiceName, defaults, "serviceName")
	setServiceStringDefault(&service.ServiceOption, defaults, "serviceOption")
	setServiceStringDefault(&service.Scope, defaults, "scope")
	setServiceStringDefault(&service.MsgName, defaults, "msgName")
	setServiceStringDefault(&service.ContainerType, defaults, "containerType")

	service.OrderType = model.NormalizeEbicsPayloadOrderType(service.OrderType)
	service.ServiceName = strings.TrimSpace(service.ServiceName)
	service.ServiceOption = strings.TrimSpace(service.ServiceOption)
	service.Scope = strings.TrimSpace(service.Scope)
	service.MsgName = strings.TrimSpace(service.MsgName)
	service.ContainerType = strings.TrimSpace(service.ContainerType)

	if service.OrderType == "" {
		return PayloadServiceRef{}, database.NewValidationError("the EBICS payload order type is missing")
	}

	return service, nil
}

func resolvePayloadTarget(
	input *PayloadRequestInput,
	profile *model.EbicsPayloadProfile,
) *PayloadTargetRef {
	if input.Target != nil && strings.TrimSpace(input.Target.Directory) != "" {
		return &PayloadTargetRef{Directory: strings.TrimSpace(input.Target.Directory)}
	}

	if profile != nil && profile.DefaultTargetDirectory != "" {
		return &PayloadTargetRef{Directory: profile.DefaultTargetDirectory}
	}

	return nil
}

func resolvePayloadFile(input *PayloadRequestInput) *PayloadFileRef {
	if input.File == nil {
		return nil
	}

	return &PayloadFileRef{
		Path:       strings.TrimSpace(input.File.Path),
		OutputName: strings.TrimSpace(input.File.OutputName),
	}
}

func mergePayloadMetadata(input, defaults map[string]any) map[string]any {
	merged := map[string]any{}

	for key, value := range defaults {
		if trimmedKey, ok := strings.CutPrefix(key, "meta."); ok {
			merged[trimmedKey] = value
		}
	}

	maps.Copy(merged, input)

	return merged
}

func extractDeclaredAmount(metadata map[string]any) (amount, currency string) {
	if metadata == nil {
		return "", ""
	}

	if rawAmount, ok := metadata["declaredAmount"]; ok {
		amount = strings.TrimSpace(fmt.Sprint(rawAmount))
	}

	if rawCurrency, ok := metadata["declaredCurrency"]; ok {
		currency = strings.ToUpper(strings.TrimSpace(fmt.Sprint(rawCurrency)))
	}

	return amount, currency
}

func setServiceStringDefault(target *string, defaults map[string]any, key string) {
	if *target != "" {
		return
	}

	raw, ok := defaults[key]
	if !ok {
		return
	}

	*target = strings.TrimSpace(fmt.Sprint(raw))
}

func resolveResolutionMode(profile *model.EbicsPayloadProfile) string {
	if profile != nil {
		return "profile"
	}

	return "free-input"
}

func resolveRuleName(
	input *PayloadRequestInput,
	profile *model.EbicsPayloadProfile,
	defaults map[string]any,
) string {
	ruleName := strings.TrimSpace(input.RuleName)
	if ruleName != "" {
		return ruleName
	}

	if profile != nil && profile.DefaultRuleName != "" {
		return strings.TrimSpace(profile.DefaultRuleName)
	}

	if raw, ok := defaults["ruleName"]; ok {
		return strings.TrimSpace(fmt.Sprint(raw))
	}

	return ""
}

func normalizePayloadSubscriber(subscriber PayloadSubscriberRef) PayloadSubscriberRef {
	return PayloadSubscriberRef{
		HostID:    strings.TrimSpace(subscriber.HostID),
		PartnerID: strings.TrimSpace(subscriber.PartnerID),
		UserID:    strings.TrimSpace(subscriber.UserID),
	}
}

func validatePayloadOrderType(orderType string) error {
	if err := model.ValidateEbicsPayloadOrderTypeForRuntime(orderType); err != nil {
		return fmt.Errorf("validate EBICS payload order type: %w", err)
	}

	return nil
}
