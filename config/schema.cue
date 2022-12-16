#Service: {
    enabled: bool
}

proxy: #Service

#Server: {
    alias:  string
    config?: string
    preset?: string
}

#ServerPreset: {
    config: string
}

#ClusterSettings: {
    #Service
    presets: [string]: #ServerPreset
    assets: [...string]
    servers: [...#Server]
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
