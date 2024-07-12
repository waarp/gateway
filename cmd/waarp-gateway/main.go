package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"

	wg "code.waarp.fr/apps/gateway/gateway/pkg/cmd/client"
)

//nolint:lll // struct tags for command line arguments can be long
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
		Add    wg.TransferAdd       `command:"add" description:"Add a new transfer to be executed"`
		Get    wg.TransferGet       `command:"get" description:"Consult a transfer"`
		List   wg.TransferList      `command:"list" description:"List the transfers"`
		Pau    wg.TransferPause     `command:"pause" description:"Pause a running transfer"`
		Res    wg.TransferResume    `command:"resume" description:"Resume a paused transfer"`
		Can    wg.TransferCancel    `command:"cancel" description:"Cancel a transfer"`
		Ret    wg.TransferRetry     `command:"retry" description:"Reprogram a canceled transfer"`
		CanAll wg.TransferCancelAll `command:"cancel-all" description:"Cancel all transfers in the given status"`
	} `command:"transfer" description:"Manage the running transfers"`

	History struct {
		Get  wg.HistoryGet   `command:"get" description:"Consult a finished transfer"`
		List wg.HistoryList  `command:"list" description:"List the finished transfers"`
		Ret  wg.HistoryRetry `command:"retry" description:"Reprogram a canceled transfer"`
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
		} `command:"monitor" description:"Manage the SNMP monitors"`
	} `command:"snmp" description:"Manage SNMP"`

	Version wg.Version `command:"version" description:"Print the program version and exit"`
}

type credCommands struct {
	Add wg.CredentialAdd    `command:"add" description:"Add a new credential"`
	Get wg.CredentialGet    `command:"get" description:"Retrieve a credential's information"`
	Del wg.CredentialDelete `command:"delete" description:"Delete a credential"`
}

type certCommands struct {
	Add  wg.CertAdd    `command:"add" description:"Add a new certificate"`
	Get  wg.CertGet    `command:"get" description:"Retrieve a certificate's information"`
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
