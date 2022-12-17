#Service: {
    enabled: bool
}

proxy: #Service

#Server: {
    alias:  string
    preset: string
}

#ServerPreset: {
    config: string
    default: bool | *false
}

#Port: >0 & <=65536 & int

#ENetIngress: {
    port:  #Port
    command: string
}
#IngressSettings: {
    desktop: [...#ENetIngress]
    web: {
        port: #Port
    }
}
#ClusterSettings: {
    #Service
    presets: [string]: #ServerPreset
    assets: [...string]
    servers: [...#Server]
    ingress: #IngressSettings
    serverDescription: string
}
cluster: #ClusterSettings

#ClientSettings: {
    #Service
    assets: [...string]
    clusters: [...string]
    proxy: string
}
client: #ClientSettings
