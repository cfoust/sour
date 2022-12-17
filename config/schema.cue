#Service: {
    enabled: bool | *true
}

proxy: #Service

#Server: {
    alias:  string
    preset: string
}

#ServerPreset: {
    name: string
    config: string
    default: bool | *false
}

#Port: uint16

#ENetIngress: {
    port:  #Port
    command: string
}
#IngressSettings: {
    desktop: [...#ENetIngress] | *[{
        port: 28785
        command: "join lobby"
    }]
    web: {
        port: #Port | *29999
    }
}
#ClusterSettings: {
    #Service
    presets: [...#ServerPreset] | *[{
        name: "default"
        default: true
        config: ##"""
        qserv_version "17"
        servermotd "Press Esc twice to change your name or adjust game options."
        maxclients 32
        defaultgamespeed 100
        defaultmodename "ffa"
        defaultmap "complex"
        autosendmap 1
        enable_passflag 1
        instacoop 0
        serverflagruns 1
        addbanner "^f7Sour ^f7is available online ^f1www.github.com/cfoust/sour^f7."
        addbanner "^f1[Tip]: ^f7Use ^f2#mapsucks ^f7to vote for an intermission."
        serverbotbalance 1
        lockmaprotation 0
        ffamaps = [
          complex dust2 turbine
        ]
        maprotationreset
        maprotation "*" $ffamaps

        maxteamkills 7
        teamkillkickreset
        teamkillkick "?capture" 10 30
        maxdemos 10
        updatemaster 0
        enablemultiplemasters 0
        serverconnectmsg 1
        welcomewithname 1
        no_single_private 0
        pingwarncustommsg "A little lagspike was detected, we just wanted to let you know."
        restrictpausegame 1
        restrictgamespeed 1
        restrictdemos 0
        """##
    }]
    assets: [...string] | *["http://localhost:1234/assets/.index.json"]
    servers: [...#Server] | *[
        {
            alias: "lobby"
            preset: "default"
        }
    ]
    ingress: #IngressSettings
    serverDescription: string | *"Sour [#id]"
}
cluster: #ClusterSettings

#ClientSettings: {
    #Service
    assets: [...string] | *["#origin/assets/.index.json"]
    clusters: [...string] | *["#host/service/cluster/"]
    proxy: string | *"#host/service/proxy/"
}
client: #ClientSettings
