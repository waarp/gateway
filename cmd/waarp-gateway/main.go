package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"

	wg "code.waarp.fr/apps/gateway/gateway/pkg/cmd/client"
)

//nolint:lll,embeddedstructfieldcheck // struct tags for command line arguments can be long
type commands struct {
	Connection struct {
		Address  wg.AddrOpt     `short:"a" long:"address" required:"yes" description:"The address of the gateway" env:"WAARP_GATEWAY_ADDRESS"`
		Insecure wg.InsecureOpt `short:"i" long:"insecure" description:"Skip certificate verification" env:"WAARP_GATEWAY_INSECURE"`
	} `group:"Connection Options" description:"The options defining how to connect to the gateway"`

	Status wg.Status `command:"status" description:"Show the status of the gateway services"`

	Server struct {
		Add         wg.ServerAdd       `command:"add" description:"Add a new server"`
		Get         wg.ServerGet       `command:"get" description:"Retrieve a server's information"`
		List        wg.ServerList      `command:"list" description:"List the known servers"`
		Upd         wg.ServerUpdate    `command:"update" description:"Update a server"`
		Del         wg.ServerDelete    `command:"delete" description:"Delete a server"`
		Auth        wg.ServerAuthorize `command:"authorize" description:"Authorize a server to use a rule"`
		Rev         wg.ServerRevoke    `command:"revoke" description:"Revoke a server's permission to use a rule"`
		Enable      wg.ServerEnable    `command:"enable" description:"Enable a server at launch"`
		Disable     wg.ServerDisable   `command:"disable" description:"Disable a server at launch"`
		Start       wg.ServerStart     `command:"start" description:"Start an offline server"`
		Stop        wg.ServerStop      `command:"stop" description:"Stop a running server"`
		Restart     wg.ServerRestart   `command:"restart" description:"Stop and restart a running server"`
		Credentials struct {
			Args struct {
				Server wg.ServerArg `required:"yes" positional-arg-name:"server" description:"The server's name"`
			} `positional-args:"yes"`
			credCommands
		} `command:"credential" description:"Manage an server's credentials"`
		Cert struct {
			Args struct {
				Server wg.ServerArg `required:"yes" positional-arg-name:"server" description:"The server's name"`
			} `positional-args:"yes"`
			certCommands
		} `command:"cert" description:"Manage an server's certificates"`
	} `command:"server" description:"Manage the local servers"`

	Client struct {
		Add     wg.ClientAdd     `command:"add" description:"Add a new client"`
		Get     wg.ClientGet     `command:"get" description:"Retrieve a client's information"`
		List    wg.ClientList    `command:"list" description:"List the known client"`
		Update  wg.ClientUpdate  `command:"update" description:"Update a client"`
		Delete  wg.ClientDelete  `command:"delete" description:"Delete a client"`
		Enable  wg.ClientEnable  `command:"enable" description:"Enable a client at launch"`
		Disable wg.ClientDisable `command:"disable" description:"Disable a client at launch"`
		Start   wg.ClientStart   `command:"start" description:"Start an offline client"`
		Stop    wg.ClientStop    `command:"stop" description:"Stop a running client"`
		Restart wg.ClientRestart `command:"restart" description:"Stop and restart a running client"`
	} `command:"client" description:"Manage the gateway's local clients"`

	Partner struct {
		Add         wg.PartnerAdd       `command:"add" description:"Add a new partner"`
		Get         wg.PartnerGet       `command:"get" description:"Retrieve a partner's information"`
		List        wg.PartnerList      `command:"list" description:"List the known partners"`
		Upd         wg.PartnerUpdate    `command:"update" description:"Update a partner"`
		Del         wg.PartnerDelete    `command:"delete" description:"Delete a partner"`
		Auth        wg.PartnerAuthorize `command:"authorize" description:"Authorize a partner to use a rule"`
		Rev         wg.PartnerRevoke    `command:"revoke" description:"Revoke a partner's permission to use a rule"`
		Credentials struct {
			Args struct {
				Partner wg.PartnerArg `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
			} `positional-args:"yes"`
			credCommands
		} `command:"credential" description:"Manage a partner's credentials"`
		Cert struct {
			Args struct {
				Partner wg.PartnerArg `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
			} `positional-args:"yes"`
			certCommands
		} `command:"cert" description:"Manage an partner's certificates"`
	} `command:"partner" description:"Manage the remote partners"`

	Account struct {
		Local struct {
			Args struct {
				Server wg.ServerArg `required:"yes" positional-arg-name:"server" description:"The server's name"`
			} `positional-args:"yes"`
			Add         wg.LocAccAdd       `command:"add" description:"Add a new local account to a server"`
			Get         wg.LocAccGet       `command:"get" description:"Retrieve a local account's information"`
			List        wg.LocAccList      `command:"list" description:"List a server's local accounts"`
			Upd         wg.LocAccUpdate    `command:"update" description:"Update a local account"`
			Del         wg.LocAccDelete    `command:"delete" description:"Delete a local account"`
			Auth        wg.LocAccAuthorize `command:"authorize" description:"Authorize a local account to use a rule"`
			Rev         wg.LocAccRevoke    `command:"revoke" description:"Revoke a local account's permission to use a rule"`
			Credentials struct {
				Args struct {
					Account wg.LocAccArg `required:"yes" positional-arg-name:"account" description:"The account's name"`
				} `positional-args:"yes"`
				credCommands
			} `command:"credential" description:"Manage a local account's credentials"`
			Cert struct {
				Args struct {
					Account wg.LocAccArg `required:"yes" positional-arg-name:"account" description:"The account's name"`
				} `positional-args:"yes"`
				certCommands
			} `command:"cert" description:"Manage a local account's certificates"`
		} `command:"local" description:"Manage a server's accounts"`
		Remote struct {
			Args struct {
				Partner wg.PartnerArg `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
			} `positional-args:"yes"`
			Add         wg.RemAccAdd       `command:"add" description:"Add a new remote account to a partner"`
			Get         wg.RemAccGet       `command:"get" description:"Retrieve a remote account's information"`
			List        wg.RemAccList      `command:"list" description:"List a partner's remote accounts"`
			Upd         wg.RemAccUpdate    `command:"update" description:"Update a remote account"`
			Del         wg.RemAccDelete    `command:"delete" description:"Delete a remote account"`
			Auth        wg.RemAccAuthorize `command:"authorize" description:"Authorize a remote account to use a rule"`
			Rev         wg.RemAccRevoke    `command:"revoke" description:"Revoke a remote account's permission to use a rule"`
			Credentials struct {
				Args struct {
					Account wg.RemAccArg `required:"yes" positional-arg-name:"account" description:"The account's name"`
				} `positional-args:"yes"`
				credCommands
			} `command:"credential" description:"Manage a remote account's credentials"`
			Cert struct {
				Args struct {
					Account wg.RemAccArg `required:"yes" positional-arg-name:"account" description:"The account's name"`
				} `positional-args:"yes"`
				certCommands
			} `command:"cert" description:"Manage a remote account's certificates"`
		} `command:"remote" description:"Manage a partner's accounts"`
	} `command:"account" description:"Manage the accounts"`

	Rule struct {
		Add  wg.RuleAdd      `command:"add" description:"Add a new rule"`
		Get  wg.RuleGet      `command:"get" description:"Retrieve a rule's information"`
		List wg.RuleList     `command:"list" description:"List the known rules"`
		Upd  wg.RuleUpdate   `command:"update" description:"Update a rule"`
		Del  wg.RuleDelete   `command:"delete" description:"Delete a rule"`
		All  wg.RuleAllowAll `command:"allow" description:"Remove all usage restriction on a rule"`
	} `command:"rule" description:"Manage the transfer rules"`

	Transfer struct {
		Add    wg.TransferAdd         `command:"add" description:"Add a new client transfer to be executed"`
		Reg    wg.TransferPreregister `command:"preregister" description:"Register a server transfer"`
		Get    wg.TransferGet         `command:"get" description:"Consult a transfer"`
		List   wg.TransferList        `command:"list" description:"List the transfers"`
		Pau    wg.TransferPause       `command:"pause" description:"Pause a running transfer"`
		Res    wg.TransferResume      `command:"resume" description:"Resume a paused transfer"`
		Can    wg.TransferCancel      `command:"cancel" description:"Cancel a transfer"`
		Ret    wg.TransferRetry       `command:"retry" description:"Reprogram a canceled transfer"`
		CanAll wg.TransferCancelAll   `command:"cancel-all" description:"Cancel all transfers in the given status"`
	} `command:"transfer" description:"Manage the running transfers"`

	History struct {
		Get  wg.HistoryGet   `command:"get" description:"Consult a finished transfer"`     //nolint:staticcheck // legacy command is still exposed for backward compatibility
		List wg.HistoryList  `command:"list" description:"List the finished transfers"`    //nolint:staticcheck // legacy command is still exposed for backward compatibility
		Ret  wg.HistoryRetry `command:"retry" description:"Reprogram a canceled transfer"` //nolint:staticcheck // legacy command is still exposed for backward compatibility
	} `command:"history" description:"Manage the transfer history [DEPRECATED: merged with the 'transfer' command] "`

	User struct {
		Add  wg.UserAdd    `command:"add" description:"Add a new user"`
		Get  wg.UserGet    `command:"get" description:"Retrieve a user's information"`
		List wg.UserList   `command:"list" description:"List the known users"`
		Upd  wg.UserUpdate `command:"update" description:"Update an existing user"`
		Del  wg.UserDelete `command:"delete" description:"Delete a user"`
	} `command:"user" description:"Manage the gateway users"`

	Override struct {
		Address struct {
			Set  wg.OverrideAddressSet    `command:"set" description:"Create or update an address indirection"`
			List wg.OverrideAddressList   `command:"list" description:"List the address indirections"`
			Del  wg.OverrideAddressDelete `command:"delete" description:"Delete an address indirection"`
		} `command:"address" description:"Manage net address indirections"`
	} `command:"override" description:"Manage the node's setting overrides"`

	Authorities struct {
		Add  wg.AuthorityAdd    `command:"add" description:"Add a new authority"`
		Get  wg.AuthorityGet    `command:"get" description:"Retrieve an authority's information"`
		List wg.AuthorityList   `command:"list" description:"List the known authorities"`
		Upd  wg.AuthorityUpdate `command:"update" description:"Update an existing authority"`
		Del  wg.AuthorityDelete `command:"delete" description:"Delete an authority"`
	} `command:"authority" description:"Manage the authentication authorities"`

	SNMP struct {
		Monitor struct {
			Add  wg.SnmpMonitorAdd    `command:"add" description:"Add a new SNMP monitor"`
			Get  wg.SnmpMonitorGet    `command:"get" description:"Retrieve a SNMP monitor's information"`
			List wg.SnmpMonitorList   `command:"list" description:"List the known SNMP monitors"`
			Upd  wg.SnmpMonitorUpdate `command:"update" description:"Update an existing SNMP monitor"`
			Del  wg.SnmpMonitorDelete `command:"delete" description:"Delete an SNMP monitor"`
			Test wg.SnmpTestMonitors  `command:"test" description:"Send a test notification to the SNMP monitors"`
		} `command:"monitor" description:"Manage the SNMP monitors"`
		Server struct {
			Set wg.SnmpServerSet    `command:"set" description:"Set the SNMP server configuration"`
			Get wg.SnmpServerGet    `command:"get" description:"Retrieve the SNMP server configuration"`
			Del wg.SnmpServerDelete `command:"delete" description:"Delete the SNMP server"`
		} `command:"server" description:"Manage the SNMP server configuration"`
	} `command:"snmp" description:"Manage SNMP"`

	Cloud struct {
		Add  wg.CloudAdd    `command:"add" description:"Add a new cloud instance"`
		Get  wg.CloudGet    `command:"get" description:"Retrieve a cloud instance's information"`
		List wg.CloudList   `command:"list" description:"List the known cloud instances"`
		Upd  wg.CloudUpdate `command:"update" description:"Update an existing cloud instance"`
		Del  wg.CloudDelete `command:"delete" description:"Delete an cloud instance"`
	} `command:"cloud" description:"Manage the gateway's cloud instances"`

	Key struct {
		Add  wg.CryptoKeysAdd    `command:"add" description:"Add a new cryptographic key"`
		Get  wg.CryptoKeysGet    `command:"get" description:"Retrieve a cryptographic key"`
		List wg.CryptoKeysList   `command:"list" description:"List the known cryptographic keys"`
		Upd  wg.CryptoKeysUpdate `command:"update" description:"Update an existing cryptographic key"`
		Del  wg.CryptoKeysDelete `command:"delete" description:"Delete a cryptographic key"`
	} `command:"key" description:"Manage the cryptographic keys"`

	Email struct {
		Template struct {
			Add  wg.EmailTemplateAdd    `command:"add" description:"Add a new email template"`
			Get  wg.EmailTemplateGet    `command:"get" description:"Retrieve an email template"`
			List wg.EmailTemplateList   `command:"list" description:"List the email templates"`
			Upd  wg.EmailTemplateUpdate `command:"update" description:"Update an existing email template"`
			Del  wg.EmailTemplateDelete `command:"delete" description:"Delete an email template"`
		} `command:"template" description:"Manage the email templates"`
		Credential struct {
			Add  wg.SMTPCredentialAdd    `command:"add" description:"Add a new SMTP credential"`
			Get  wg.SMTPCredentialGet    `command:"get" description:"Retrieve an SMTP credential"`
			List wg.SMTPCredentialList   `command:"list" description:"List the SMTP credentials"`
			Upd  wg.SMTPCredentialUpdate `command:"update" description:"Update an existing SMTP credential"`
			Del  wg.SMTPCredentialDelete `command:"delete" description:"Delete an SMTP credential"`
		} `command:"credential" description:"Manage the SMTP credentials"`
	} `command:"email" description:"Manage the gateway's email configuration"`

	Ebics struct {
		Operation struct {
			List      wg.EbicsOperationList      `command:"list" description:"List the EBICS operations"`
			Get       wg.EbicsOperationGet       `command:"get" description:"Retrieve an EBICS operation"`
			Reporting wg.EbicsOperationReporting `command:"reporting" description:"Execute an EBICS reporting order"`
			Signature wg.EbicsOperationSignature `command:"signature" description:"Execute an EBICS signature order"`
		} `command:"operation" description:"Manage the EBICS operations"`

		Transaction struct {
			List     wg.EbicsTransactionList       `command:"list" description:"List the EBICS transactions"`
			Get      wg.EbicsTransactionGet        `command:"get" description:"Retrieve an EBICS transaction"`
			Segments wg.EbicsTransactionSegments   `command:"segments" description:"List the segments of an EBICS transaction"`
			Segment  wg.EbicsTransactionSegmentGet `command:"segment" description:"Retrieve one EBICS transaction segment"`
		} `command:"transaction" description:"Manage the EBICS transactions"`

		Payload struct {
			Upload   wg.EbicsPayloadUpload   `command:"upload" description:"Submit an EBICS payload upload"`
			Download wg.EbicsPayloadDownload `command:"download" description:"Submit an EBICS payload download"`
			List     wg.EbicsPayloadList     `command:"list" description:"List the EBICS payload operations"`
			Get      wg.EbicsPayloadGet      `command:"get" description:"Retrieve an EBICS payload operation"`
			Retry    wg.EbicsPayloadRetry    `command:"retry" description:"Retry an EBICS payload operation"`
			Recover  wg.EbicsPayloadRecover  `command:"recover" description:"Recover an EBICS payload operation"`
			Profile  struct {
				Add    wg.EbicsPayloadProfileAdd    `command:"add" description:"Add an EBICS payload profile"`
				List   wg.EbicsPayloadProfileList   `command:"list" description:"List the EBICS payload profiles"`
				Get    wg.EbicsPayloadProfileGet    `command:"get" description:"Retrieve an EBICS payload profile"`
				Update wg.EbicsPayloadProfileUpdate `command:"update" description:"Update an EBICS payload profile"`
				Delete wg.EbicsPayloadProfileDelete `command:"delete" description:"Delete an EBICS payload profile"`
			} `command:"profile" description:"Manage the EBICS payload profiles"`
		} `command:"payload" description:"Manage the EBICS payload operations"`

		ContractView struct {
			List    wg.EbicsContractViewList    `command:"list" description:"List the EBICS contract views"`
			Get     wg.EbicsContractViewGet     `command:"get" description:"Retrieve an EBICS contract view"`
			Refresh wg.EbicsContractViewRefresh `command:"refresh" description:"Refresh EBICS contract views from the bank"`
		} `command:"contract-view" description:"Manage the EBICS contract views"`

		KeyLifecycle struct {
			List   wg.EbicsKeyLifecycleList   `command:"list" description:"List the EBICS key lifecycles"`
			Get    wg.EbicsKeyLifecycleGet    `command:"get" description:"Retrieve an EBICS key lifecycle"`
			Action wg.EbicsKeyLifecycleAction `command:"action" description:"Apply an action to an EBICS key lifecycle"`
		} `command:"key-lifecycle" description:"Manage the EBICS key lifecycles"`

		Initialization struct {
			List   wg.EbicsInitializationList   `command:"list" description:"List the EBICS initializations"`
			Get    wg.EbicsInitializationGet    `command:"get" description:"Retrieve an EBICS initialization workflow"`
			Action wg.EbicsInitializationAction `command:"action" description:"Apply an action to an EBICS initialization workflow"`
		} `command:"initialization" description:"Manage the EBICS initialization workflows"`

		KeyRotation struct {
			Prepare wg.EbicsKeyRotationPrepare `command:"prepare" description:"Prepare a coordinated EBICS key rotation"`
			Send    wg.EbicsKeyRotationSend    `command:"send" description:"Send a coordinated EBICS key rotation to the bank"`
			Confirm wg.EbicsKeyRotationConfirm `command:"confirm" description:"Confirm a coordinated EBICS key rotation"`
			Cancel  wg.EbicsKeyRotationCancel  `command:"cancel" description:"Cancel a coordinated EBICS key rotation"`
			Reject  wg.EbicsKeyRotationReject  `command:"reject" description:"Reject a coordinated EBICS key rotation"`
			Revoke  wg.EbicsKeyRotationRevoke  `command:"revoke" description:"Revoke a coordinated EBICS key rotation with SPR"`
		} `command:"key-rotation" description:"Manage coordinated EBICS key rotations"`

		RTN struct {
			Provider struct {
				Add    wg.EbicsRTNProviderAdd    `command:"add" description:"Add an EBICS RTN provider"`
				List   wg.EbicsRTNProviderList   `command:"list" description:"List the EBICS RTN providers"`
				Get    wg.EbicsRTNProviderGet    `command:"get" description:"Retrieve an EBICS RTN provider"`
				Update wg.EbicsRTNProviderUpdate `command:"update" description:"Update an EBICS RTN provider"`
				Delete wg.EbicsRTNProviderDelete `command:"delete" description:"Delete an EBICS RTN provider"`
			} `command:"provider" description:"Manage the EBICS RTN providers"`
			Event struct {
				List       wg.EbicsRTNEventList       `command:"list" description:"List the EBICS RTN events"`
				Get        wg.EbicsRTNEventGet        `command:"get" description:"Retrieve an EBICS RTN event"`
				Retry      wg.EbicsRTNEventRetry      `command:"retry" description:"Retry an EBICS RTN event"`
				Quarantine wg.EbicsRTNEventQuarantine `command:"quarantine" description:"Quarantine an EBICS RTN event"`
			} `command:"event" description:"Manage the EBICS RTN events"`
		} `command:"rtn" description:"Manage the EBICS RTN"`

		RuntimePolicy struct {
			Get    wg.EbicsRuntimePolicyGet    `command:"get" description:"Retrieve the EBICS runtime policy"`
			Update wg.EbicsRuntimePolicyUpdate `command:"update" description:"Update the EBICS runtime policy"`
		} `command:"runtime-policy" description:"Manage the EBICS runtime policy"`
	} `command:"ebics" description:"Manage EBICS"`

	Version wg.Version `command:"version" description:"Print the program version and exit"`
}

type credCommands struct {
	Add wg.CredentialAdd    `command:"add" description:"Add a new credential"`
	Get wg.CredentialGet    `command:"get" description:"Retrieve a credential's information"`
	Del wg.CredentialDelete `command:"delete" description:"Delete a credential"`
}

type certCommands struct {
	Add wg.CertAdd `command:"add" description:"Add a new certificate"`
	//nolint:staticcheck // legacy certificate commands are kept for compatibility
	Get wg.CertGet `command:"get" description:"Retrieve a certificate's information"`
	//nolint:staticcheck // legacy certificate commands are kept for compatibility
	List wg.CertList   `command:"list" description:"List the known certificates"`
	Upd  wg.CertUpdate `command:"update" description:"Update a certificate"`
	Del  wg.CertDelete `command:"delete" description:"Delete a certificate"`
}

func main() {
	parser := flags.NewNamedParser("waarp-gateway", flags.Default)
	parser.Usage = "[CONNECTION-OPTIONS]"

	cmd := &commands{}
	cmd.Connection.Insecure = wg.SetInsecureFlag

	if err := wg.InitParser(parser, cmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := wg.Main(parser, os.Args[1:]); err != nil {
		// fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
