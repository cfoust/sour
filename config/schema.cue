#Discord: {
	enabled: bool | *false
	// The domain used for desktop client keys -- should be unique to your
	// instance
	domain:           string | *"sour"
	id:               string | *""
	secret:           string | *""
	redirectURI:      string | *""
	authorizationURL: "https://discord.com/api/oauth2/authorize?client_id=\(id)&redirect_uri={{redirectURI}}&response_type=code&scope=identify&prompt=none"
}

discord: #Discord

redis: {
	address:  string | *"localhost:6379"
	password: string | *""
	DB:       int | *0
}

#Service: {
	enabled: bool | *true
}

proxy: #Service

#SpaceLink: {
	id:          uint8
	destination: string
}

#SpaceConfig: {
	alias:       string
	description: string | *""
	links: [...#SpaceLink]
}

#Space: {
	// An alias replaces the ID of a space. For example, users could run
	// #join lobby (on desktop) or /join lobby (on the web)
	preset: string
	// Whether voting on a map should create a game.
	votingCreates: bool | *true
	// In explore mode, change the map every 3 minutes.
	exploreMode: bool | *false
	// Skip maps in this root when in explore mode
	exploreModeSkip: string | *""
	config:          #SpaceConfig
}

#ServerConfig: {
	maxClients: uint8 | *128
	// Length of game in seconds
	matchLength:      uint | *600
	defaultGameSpeed: uint8 | *100
	defaultMode:      "ffa" | "coop" | "insta" | "instateam" | "effic" | "efficteam" | "tac" | "tacteam" | "ctf" | "instactf" | "efficctf" | *"ffa"
	defaultMap:       string | *"complex"
	maps:             [...string] | *[]
}

#ServerPreset: {
	name: string

	// If this is true, the user cannot create servers with this preset,
	// it's only for inheritance or matchmaking purposes.
	virtual: bool | *false

	config: #ServerConfig

	// This means that if the user does not specify a config, we use this preset.
	default: bool | *false
}

#DuelType: {
	name:   string
	preset: string
	// How long the warmup lasts
	warmupSeconds: uint | *30
	// How long the main match (after warmup lasts)
	gameSeconds: uint | *180
	// How many frags a player has to win by to win
	// (otherwise the game goes into overtime)
	winThreshold: uint | *3
	// The length of each overtime session. If a player is still not
	// winning by winThreshold, overtime is repeated.
	overtimeSeconds: uint | *60
	// "all" = force respawn everyone when someone dies
	// "dead" = force respawn just the person who died when they die
	// "none" = don't respawn anyone
	forceRespawn: "all" | "dead" | "none" | *"all"
	pauseOnDeath: bool | *false
	default:      bool | *false
}

#MatchmakingSettings: {
	duel: [...#DuelType]
}

#Port: uint16

#ENetIngress: {
	// The UDP port to listen on
	port: #Port
	// The name of the server to join when a client connects
	target: string
	// Configure the serverinfo port
	serverInfo: {
		enabled: bool | *false
		// Whether to register with the master server
		master: bool | *false
		// Whether to use the cluster's server info instead of the target info
		cluster: bool | *false
	}
}

// Describes all of the ways desktop clients can join this cluster
#IngressSettings: {
	desktop: [...#ENetIngress]
	web: {
		// The TCP port the WebSocket service should listen on.
		port: #Port
	}
}

#StoreConfig: {
	type: "fs"
	path: string
}

#AssetStore: {
	// Used in the `location` field of assets in the database.
	name:    string
	default: bool | *false
	config:  #StoreConfig
}

// Storage locations for user-provided assets, including maps.
assetStores: [...#AssetStore] | *[{
	name:    "default"
	default: true
	config: {
		type: "fs"
		path: "./assets"
	}
}]

// Sour clusters host game servers.
#ClusterSettings: {
	#Service

	dbPath: string | *"state.db"

	// Whether to save demos of user sessions to Redis.
	logSessions: bool | *false

	// If set, saves server logs to this directory.
	logDirectory: string | *""

	// If set, used for caching assets in lieu of Redis.
	cacheDirectory: string | *"/tmp/assets"

	// Information used to respond to server info requests
	serverInfo: {
		map:         string | *"Sourland"
		description: string | *"Sour"
		timeLeft:    uint | *3600
		gameSpeed:   uint | *100
	}

	// The message of the day for the server, sent to clients on join.
	serverMOTD: string | *"Press Esc twice to change your name or adjust game options."

	banners: [...string] | *[
			"^f7Sour ^f7is available online ^f1www.github.com/cfoust/sour^f7.",
			"Queue for duels with #duel or #duel insta.",
	]
	// How often to send a banner (seconds)
	bannerInterval: uint32 | *100

	// Server presets are templates for starting new servers, typically by a user through #creategame.
	// A mapping from preset name -> preset settings.
	presets: [...#ServerPreset]

	// This is not the same thing as client.assets because the cluster has to
	// specify complete URLs (and can access services using their addresses
	// inside the container.)
	assets: [...string]

	// These are all of the game servers that will be started when the cluster starts.
	spaces: [...#Space]

	matchmaking: #MatchmakingSettings
	ingress:     #IngressSettings

	// We set the Sauerbraten `serverdesc` according to this template.
	// #id is replaced with the server's identifier.
	serverDescription: string | *"Sour [#id]"
}
cluster: #ClusterSettings

#ClientSettings: {
	#Service

	auth: {
		enabled:          discord.enabled
		authorizationURL: discord.authorizationURL
		redirectURI:      discord.redirectURI
		domain:           discord.domain
	}

	// All client URLs can use these template variables:
	// #host: replaced with window.location.host
	// #origin: replaced with window.location.origin (basically #protocol + #host)
	// #protocol: replaced with window.location.protocol e.g. https:

	// These are the URLs for each of the asset sources.
	// Order matters; the client uses the first map it finds.
	// The client's asset sources may not be the same as the cluster's because
	// we might not know the hostname the user will be accessing Sour at ahead
	// of time. We can take advantage of the browser's automatic addition of the
	// hostname to bare absolute paths.
	assets: [...string] | *["#origin/assets/.index.source"]

	// The URLs for all of the game servers, for now we only support one.
	// ws: and wss: are inferred
	clusters: [...string] | *["#host/service/cluster/"]
	// ws: and wss: are inferred
	proxy: string | *"#host/service/proxy/"

	menuOptions: string | *"guibutton \"play\" \"join\""
}
client: #ClientSettings
