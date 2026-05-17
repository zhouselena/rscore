# Game Infrastructure Outage Analysis — 10 Multiplayer Games
## Pre- and Post-Outage Node/Edge CSV Summary

Each game has four CSVs:
- `GameName_pre_nodes.csv` / `GameName_pre_edges.csv` — infrastructure before the outage
- `GameName_post_nodes.csv` / `GameName_post_edges.csv` — infrastructure after remediation

---

## Schema

### Nodes: `Node, Type, Service Tier`

| Field | Values |
|---|---|
| `Type` | `Functional` — an application feature or service owned by the game developer |
| `Type` | `Provider` — a cloud, hosting, or third-party infrastructure provider |
| `Service Tier` | Populated for `Provider` nodes only, matching the `Service/Tier` column in `providers.csv`. Left blank for `Functional` nodes and for providers not listed in `providers.csv` (e.g. on-prem DCs, HashiCorp tools, Photon, Steam/Valve, Xbox Live, PSN, Azure PlayFab, Azure CDN, Azure Cache for Redis) |

### Edges: `Type, From, To`

| Edge Type | Direction | Meaning |
|---|---|---|
| `Hosted-on` | Functional → Provider | This service is deployed on / backed by this provider |
| `Dependency` | Provider → Functional | This provider delivers to / enables this downstream service |
| `Dependency` | Functional → Functional | Two services call each other directly via API, with no provider intermediary |

**Provider → Provider edges are never used.** Providers are always leaf nodes in terms of outgoing connectivity — they only point to Functional nodes.

### How Resilience Is Encoded in the Graph

Resilience is expressed through **path multiplicity** between functional nodes. To get from Functional node A to Functional node B, traffic must flow through at least one provider:

```
Functional A  --[Hosted-on]-->  Provider  --[Dependency]-->  Functional B
```

If only one provider sits between A and B, there is one path and one point of failure — low resilience. If multiple independent providers each connect A to B, there are multiple paths — high resilience. For example:

```
# Low resilience — single path:
Matchmaking Service  -->  AWS EC2 (single AZ)  -->  Game Server

# High resilience — three independent paths:
Matchmaking Service  -->  AWS EKS (Fargate multi-AZ)  -->  Game Server
Matchmaking Service  -->  AWS Global Accelerator       -->  Game Server
Matchmaking Service  -->  Aurora Global Database       -->  Game Server
```

Shared providers (e.g. a database used by both Auth and Inventory) have outgoing `Dependency` edges to all functional nodes that consume them, accurately reflecting that one provider failure can affect multiple downstream services simultaneously.

Direct Functional → Functional `Dependency` edges are used only where two services have a genuine direct API call relationship with no provider in between (e.g. an auth service validating a token inline before another service proceeds).

---

## Programmatic Integrity Guarantees

All checks below were validated against every file before release:

| Check | Result |
|---|---|
| Single connected component per graph | ✓ All 20 graphs pass |
| All edge `From`/`To` names match a node in the same file | ✓ All 40 files pass |
| No `From`/`To` pair appears as both `Hosted-on` and `Dependency` | ✓ All 40 files pass |
| No exact duplicate edges (same Type, From, To) | ✓ All 40 files pass |
| No Provider → Provider edges | ✓ All 40 files pass |
| All `Hosted-on` edges are Functional → Provider | ✓ All 40 files pass |
| All non-blank `Service Tier` values exist in `providers.csv` | ✓ All 40 files pass |

### Blank Service Tiers (Expected)

46 Provider nodes intentionally have blank `Service Tier` values because they are not listed in `providers.csv`. These fall into four categories:

- **Self-hosted / on-prem:** Roblox On-Prem DCs, Mojang Auth Server, OSRS London DC, OSRS Secondary DC, INAP Dedicated Hosting, Managed Dedicated Servers (post-INAP), On-Prem MySQL
- **Proprietary game middleware:** Photon Cloud (Exit Games), Custom Hazel .NET Server, HashiCorp Consul, HashiCorp Nomad, HashiCorp Vault
- **Platform auth & identity (not in providers.csv):** Xbox Live / Microsoft Account, PlayStation Network, Steam / Valve Infrastructure, Azure Active Directory B2C
- **Azure services not in providers.csv:** Azure PlayFab, Azure CDN, Azure Cache for Redis (single-region), Azure Cache for Redis (zone-redundant), Azure Data Explorer
- **Generic legacy compute:** VPS Texas (NA), VPS Germany (EU), VPS Japan (Asia), VPS UK (single region)

---

## 1. Fortnite (Epic Games)

### Outage
**April 11–12, 2018 — MongoDB Account Service Collapse (22-hour outage)**

A new API call pattern introduced in patch 3.5 slowly degraded the MongoDB cache backing the Account Service, causing page evictions and replica set leader elections that cascaded into a complete authentication failure. All logins were blocked globally for ~22 hours. Epic attempted restoration 7 times before succeeding.

### Sources
- **Official Postmortem (4/11/2018):** https://www.epicgames.com/fortnite/en-US/news/postmortem-of-service-outage-4-12
- **Official Postmortem (3.4M CCU):** https://www.epicgames.com/fortnite/en-US/news/postmortem-of-service-outage-at-3-4m-ccu
- **Epic Public Status:** https://status.epicgames.com/
- **Fortnite Insider summary:** https://fortniteinsider.com/postmortem-of-service-outage/

### Infrastructure Changes (Pre → Post)
| Component | Pre-Outage | Post-Outage |
|---|---|---|
| Account DB | MongoDB Atlas M30+ replica set (single-region) | MongoDB Atlas Global Clusters (multi-region sharded) |
| Auth Cache | ElastiCache single node | ElastiCache cluster mode |
| Game Servers | EC2 M/C/R class single AZ | EC2 Auto Scaling Group multi-AZ |
| Backend Containers | ECS/EKS EC2-backed multi-AZ | ECS/EKS Fargate multi-AZ |
| MCP Database | RDS Multi-AZ Standby | Aurora Provisioned single region (faster ~30s failover) |
| Messaging | XMPP/Nginx (saturated under load) | Kinesis multi-shard enhanced fanout |

### Why the Post Graph Is More Resilient
In the pre-outage graph, the Account Service has only one database provider (MongoDB M30+ replica set) and one cache provider (ElastiCache single node) mediating access to downstream services — a single cache pressure spike on either collapses all paths. The post-outage graph adds MongoDB Global Clusters (distributes reads across shards and regions), clustered ElastiCache (eliminates the single-node SPOF), and Aurora replacing the slower-failover RDS standby. The number of independent provider paths between Account Service and its consumers increases, as does the resilience score of each provider. Kinesis replaces the XMPP stack that saturated Nginx worker threads under load.

---

## 2. Roblox

### Outage
**October 28–31, 2021 — HashiStack Consul Single-Cluster Collapse (73-hour outage)**

A BoltDB disk-leak bug combined with a newly activated Consul streaming feature caused write contention on a single Go channel inside Consul, degrading write latency from ~300ms to ~2 seconds. Because Nomad and Vault both depend on Consul for service discovery and secret retrieval, all 18,000+ servers and 170,000 containers became unable to communicate. Monitoring tools also went offline because they shared the same Consul cluster. 50 million users were affected.

### Sources
- **Official Roblox Postmortem:** https://about.roblox.com/newsroom/2022/01/roblox-return-to-service-10-28-10-31-2021
- **Pingdom Analysis:** https://www.pingdom.com/blog/the-roblox-outage/
- **InfoWorld Deep Dive:** https://www.infoworld.com/article/2334415/robloxs-cloud-native-catastrophe-a-post-mortem.html
- **Data Center Dynamics (new DC added):** https://www.datacenterdynamics.com/en/news/roblox-adds-data-center-after-73-hour-outage/

### Infrastructure Changes (Pre → Post)
| Component | Pre-Outage | Post-Outage |
|---|---|---|
| Consul | Single cluster, streaming enabled, BoltDB unfixed | Multi-cluster across two geo-distinct DCs; streaming disabled; BoltDB patched |
| Data Centers | Single primary on-prem DC | Primary DC + new geographically distinct secondary DC |
| Nomad | Single Consul-dependent cluster | Upgraded Nomad depending on distributed Consul |
| Telemetry | Shared same Consul cluster (went dark) | Decoupled with deeper Consul/BoltDB visibility |

### Why the Post Graph Is More Resilient
In the pre-outage graph, HashiCorp Consul (single cluster) is the sole provider mediating paths between every functional node and its dependencies — a bottleneck with only one outgoing path. Consul's failure cuts every provider-to-functional edge in the graph simultaneously. The post-outage graph replaces it with a multi-cluster Consul distributed across two geo-distinct DCs, so no single cluster failure can sever all paths. The on-prem primary DC also gains a secondary, giving Game Server and Auth two independent compute paths.

---

## 3. Among Us (Innersloth)

### Outage
**September 2020 — Viral Surge Overwhelms Custom Hazel Servers + March 2022 DDoS**

Among Us exploded from ~1,000 daily players to 500 million downloads in months. The custom Hazel .NET server with no auto-scaling and fixed-capacity VPS instances (US/EU/Asia) became completely overwhelmed — capacity had to be added manually in near-real-time. In March 2022 a DDoS attack targeting Photon relay infrastructure took servers offline entirely.

### Sources
- **Developer blog (precursor):** https://innersloth.itch.io/among-us/devlog/61029/server-issues-and-a-new-update
- **Developer Twitter thread (Sept 2020):** https://gamewatcher.com/news/among-us-server-status-maintenance-offline
- **Unity Case Study (migration to Unity Multiplay):** https://unity.com/case-study/innersloth-among-us
- **PC Gamer DDoS report (March 2022):** https://www.pcgamer.com/among-us-servers-taken-offline-after-ddos-attack/
- **Innersloth server info:** https://innersloth.zendesk.com/hc/en-us/articles/9686064498580-About-the-Among-Us-servers
- **Innersloth data blog:** https://www.innersloth.com/the-data-among-us/

### Infrastructure Changes (Pre → Post)
| Component | Pre-Outage | Post-Outage |
|---|---|---|
| Game Servers | Custom Hazel .NET on fixed-capacity VPS (no auto-scale) | GCP Cloud Run via Unity Multiplay (auto-scaled) |
| Auth | Anonymous/Hazel-integrated | Auth0 B2C via Unity Authentication |
| Analytics | None | GCP Firestore via Unity Analytics |
| DDoS Protection | None | Cloudflare Business on public endpoints |

### Why the Post Graph Is More Resilient
In the pre-outage graph, Game Session Server is reached only through Custom Hazel .NET Server and three fixed-capacity VPS nodes — none of which can auto-scale, and all of which become saturated under viral load. The post-outage graph replaces this with GCP Cloud Run (auto-scaled) plus three legacy VPS fallback paths, giving Game Session Server four independent provider paths. Cloudflare Business sits in front of Lobby & Matchmaking Service as an additional protective provider layer, blocking the DDoS vector that took the March 2022 service down.

---

## 4. Valorant (Riot Games)

### Outage / Migration Event
**2019–2023 — On-Premises Data Centers → Full AWS EKS Migration**

Valorant launched in 2020 with a hybrid on-prem/cloud model. On-prem data centers in Las Vegas, Chile, and Brazil hosted game servers while AWS handled backend services. Hardware failures on-prem caused 90-minute outages with no auto-failover. Riot progressively migrated to AWS EKS (246 clusters managed by Karpenter) and closed all 16 data centers by early 2024, achieving a $10M annual cost saving and a 35ms RTT SLA via AWS Local Zones.

### Sources
- **AWS Case Study (EKS migration):** https://aws.amazon.com/solutions/case-studies/riot-games-case-study/
- **AWS Blog (final DC decommission):** https://aws.amazon.com/blogs/gametech/riot-games-prepares-to-close-its-last-data-center-as-it-completes-global-migration-to-aws/
- **AWS RDS/Aurora Case Study:** https://aws.amazon.com/solutions/case-studies/riot-games-rds-case-study/
- **AWS x Riot Tech Panel (Local Zones):** https://medium.com/@chloemcateer/aws-x-riot-games-valorant-experience-eb42be4ba0e0
- **Riot Tech Blog:** https://technology.riotgames.com/tags/infrastructure

### Infrastructure Changes (Pre → Post)
| Component | Pre-Outage | Post-Outage |
|---|---|---|
| Game Servers | On-prem DCs (Las Vegas, Chile, Brazil) — no auto-failover | ECS/EKS Fargate multi-AZ + AWS Global Accelerator |
| Auth / Player Platform | RDS Multi-AZ Standby + EC2 single AZ | Aurora Global Database |
| Backend Containers | EC2 single AZ | ECS/EKS Fargate multi-AZ |
| Patch Delivery | CloudFront CDN | CloudFront CDN (unchanged) |

### Why the Post Graph Is More Resilient
In the pre-outage graph, Game Server is reached through three regionally isolated on-prem DCs with no failover between them — a hardware failure in any single DC severs that region's path entirely. The post-outage graph routes Game Server through both ECS/EKS Fargate multi-AZ and AWS Global Accelerator, giving two independent high-availability provider paths. Authentication Service gains Aurora Global Database (cross-region RTO < 1 min) alongside Fargate multi-AZ, replacing the single-path EC2 + RDS Standby combination.

---

## 5. Apex Legends (Respawn / EA)

### Outage / Incident
**March 18, 2024 — ALGS Regional Finals RCE Hack (competitive integrity breach)**

During the live ALGS NA Pro League Finals, a hacker exploited a remote code execution vulnerability in the Apex Legends game client to inject aimbot and wallhack cheats into the games of professional players Genburten and ImperialHal mid-match. Respawn shut down the tournament and deployed a "first of a layered series" of security patches starting March 20, 2024, confirmed patched by March 26.

### Sources
- **Dexerto investigation:** https://www.dexerto.com/apex-legends/massive-apex-legends-hack-explained-algs-impact-culprit-more-2595554/
- **GameSpot / EAC statement:** https://www.gamespot.com/articles/apex-legends-streamers-hacked-during-algs-tournament-and-the-suspected-cause-is-ironic-update/1100-6521923/
- **PC Games N live coverage:** https://www.pcgamesn.com/apex-legends/algs-finals-2024-hack
- **Gameriv (exploit patched):** https://gameriv.com/algs-hacker-reveals-the-exploit-he-used-in-tournament-has-been-patched/
- **win.gg (layered security update):** https://win.gg/news/apex-legends-hack-update-respawn-adds-a-layered-security/

### Infrastructure Changes (Pre → Post)
| Component | Pre-Outage | Post-Outage |
|---|---|---|
| Anti-Cheat | Easy Anti-Cheat only; no process isolation | EAC + Respawn layered security; sandboxed player-process isolation |
| Auth Cache | ElastiCache single node | ElastiCache cluster mode |
| Account DB | RDS Multi-AZ Standby | RDS Multi-AZ Cluster (2 readable standbys) |
| Public Endpoints | No WAF | Cloudflare Enterprise WAF + DDoS protection |

### Why the Post Graph Is More Resilient
In the pre-outage graph, EA Account Service is backed by a single-node ElastiCache and RDS Multi-AZ Standby — one cache failure removes the only fast-path session lookup. Game Server is also reached through a single compute provider (EC2 Auto Scaling Group) with no edge-layer protection. The post-outage graph adds Cloudflare Enterprise as a second provider path into Game Server (absorbing external attack traffic before it reaches EC2), upgrades the account DB to RDS Multi-AZ Cluster with faster failover, and replaces the single-node cache with clustered ElastiCache. Easy Anti-Cheat + Layered Security becomes a mandatory provider-level gate that Game Server depends on before accepting player input.

---

## 6. Minecraft / Mojang (Microsoft)

### Outage / Migration Event
**2021–2023 — Mojang Account → Microsoft Account Forced Migration**

Mojang's legacy authentication system ran on Mojang-operated infrastructure separate from Microsoft Azure, with no 2FA and a single point of failure for session validation. Surges during the mandatory migration period caused auth outages. The migration completed September 19, 2023, moving all Java Edition players to Microsoft Accounts backed by Azure AD B2C with 2FA, parental controls, and multi-region database redundancy.

### Sources
- **Minecraft.net migration announcement:** https://www.minecraft.net/en-us/article/last-call-voluntarily-migrate-java-accounts
- **Minecraft Wiki migration details:** https://minecraft.wiki/w/Java_account_migration
- **gHacks migration explainer:** https://www.ghacks.net/2022/02/06/minecraft-requires-a-microsoft-account-from-march-2022-onward/
- **Azure outage Minecraft impact (Oct 2025):** https://legalunitedstates.com/minecraft-servers-down/

### Infrastructure Changes (Pre → Post)
| Component | Pre-Outage | Post-Outage |
|---|---|---|
| Authentication | Mojang Auth Server (self-hosted, no 2FA) | Azure Active Directory B2C (2FA, enterprise SLA) |
| Player Profiles DB | Azure SQL General Purpose | Azure Cosmos DB multi-region writes |
| Account Security | No 2FA, basic password | 2FA + Microsoft Family Safety |
| Parental Controls | Minimal | Full Microsoft Family Safety integration |

### Why the Post Graph Is More Resilient
In the pre-outage graph, Mojang Authentication Service and Session Server are both backed solely by the self-hosted Mojang Auth Server — a single provider with no redundancy that must be reached before Multiplayer Game Servers can function. The post-outage graph replaces this with Azure Active Directory B2C backing both Microsoft Account Authentication and Session Server, while Profile & Username Service gains a second provider path through Azure Cosmos DB multi-region writes alongside Azure App Service. Cosmos DB's multi-region active-active writes mean the profile data path survives a regional failure.

---

## 7. Old School RuneScape / RuneScape (Jagex)

### Outage
**November 22, 2022 — London Data Center Power Failure (17-hour outage)**

An external data center provider in London suffered a site-wide power failure, taking both RuneScape and Old School RuneScape completely offline for approximately 17 hours — the longest outage in OSRS history. All game worlds, the website, HiScores, and account services went down simultaneously because all Jagex production infrastructure was co-located in a single external data center.

### Sources
- **OSRS Wiki official statement:** https://oldschool.runescape.wiki/w/Update:Recent_Game_Outages_-_Nov_22_%26_23
- **Massively Overpowered outage report:** https://massivelyop.com/2022/11/23/runescape-and-old-school-runescape-return-to-service-after-lengthy-outage/
- **GamesRadar (longest outage):** https://gamesradar.com/one-of-the-oldest-active-mmos-just-had-its-longest-server-outage-ever

### Infrastructure Changes (Pre → Post)
| Component | Pre-Outage | Post-Outage |
|---|---|---|
| Data Centers | Single external London DC | Primary London DC + new secondary UK DC |
| Game/Service DB | On-Prem MySQL single DC | AWS RDS Multi-AZ Standby |
| Auth Servers | Single London DC | Redundant instances across both DCs |
| Web / API | Single London DC | Cloudflare Business fronting web services |

### Why the Post Graph Is More Resilient
In the pre-outage graph, the External London Data Center (single provider) is the only provider with outgoing edges to every functional node — a true single point of failure. One power outage severs all provider-to-functional paths simultaneously. The post-outage graph adds a Secondary UK DC as a second compute path, AWS RDS Multi-AZ Standby as an independent database path, and Cloudflare Business fronting RuneScape Web & News. Game World Servers now have three independent provider paths (primary DC, secondary DC, RDS), meaning the graph retains connectivity even if the primary DC is completely lost.

---

## 8. Hypixel (Hypixel Inc / Riot)

### Outage
**June–July 2021 — Volumetric DDoS via INAP Provider (repeated multi-day outages)**

Hypixel suffered repeated severe DDoS attacks exploiting vulnerabilities in its upstream provider INAP (Internap). Because Hypixel's servers were directly exposed through INAP without enterprise-grade DDoS absorption, the attacks caused multi-day outages affecting tens of millions of players. A subsequent May 2022 DNS poisoning attack ("Haxickle") demonstrated continued exposure through unproxied DNS records.

### Sources
- **Hypixel Forum DDoS analysis:** https://hypixel.net/threads/hypixels-biggest-ddos-attack-solved-why-what-when.4449653/
- **Cloudflare Q2 2021 DDoS report:** https://blog.cloudflare.com/ddos-attack-trends-for-2021-q2/
- **Haxickle DNS attack analysis:** https://hypixel.net/threads/the-haxickle-accident-a-thorough-description-of-the-events.4934740/
- **Hypixel Status Page (August 2022 upstream outage):** https://status.hypixel.net/incidents/1zxt4zcbrp1t
- **Nixinova News (2021 DDoS):** https://news.nixinova.com/news/2021/06/hypixel-back-online-after-large-scale-ddos-attacks

### Infrastructure Changes (Pre → Post)
| Component | Pre-Outage | Post-Outage |
|---|---|---|
| DDoS Protection | None (INAP-exposed public IPs) | Cloudflare Enterprise (anycast, WAF, proxied DNS) |
| Hosting Provider | INAP co-location (shared DDoS target) | New colo provider with improved upstream diversity |
| Session Cache | ElastiCache single node | ElastiCache cluster mode |
| Public DNS | Direct INAP-routed (poisonable) | Cloudflare-proxied |

### Why the Post Graph Is More Resilient
In the pre-outage graph, INAP Dedicated Hosting is the sole provider with outgoing edges to all functional nodes — making it both a single point of failure and a DDoS exposure surface. The post-outage graph replaces INAP with Managed Dedicated Servers and adds Cloudflare Enterprise as a second independent provider path into Lobby & Hub Service, Hypixel Store, Public API, and Forums & Community Website. Cloudflare absorbs volumetric DDoS traffic before it reaches origin servers, and the origin IP addresses are hidden behind Cloudflare's anycast network.

---

## 9. Sea of Thieves (Rare / Xbox)

### Outage / Migration Event
**March 2018 Launch — Azure VM Instability Under Launch Traffic**

Sea of Thieves launched March 20, 2018 to massive demand that overwhelmed early single-instance Azure VMs with no availability zone redundancy or auto-scaling, causing repeated login failures and session crashes in the first weeks. Over subsequent seasons Rare migrated to AKS (containerized game servers with live migration), PlayFab (matchmaking, economy, server orchestration), Cosmos DB (replacing Azure SQL), and Azure Data Explorer (real-time telemetry).

### Sources
- **Microsoft Developer Stories (PlayFab/AKS):** https://learn.microsoft.com/en-us/shows/developer-stories/sea-of-thieves
- **Sea of Thieves status monitoring:** https://statusgator.com/services/sea-of-thieves
- **Azure outage postmortem context:** https://www.techtarget.com/searchcloudcomputing/blog/The-Troposphere/Learn-from-these-Microsofts-Azure-outage-postmortem-takeaways

### Infrastructure Changes (Pre → Post)
| Component | Pre-Outage | Post-Outage |
|---|---|---|
| Game Servers | Azure VM single instance (no zone redundancy) | Azure Kubernetes Service (AKS) with live player migration |
| Matchmaking | Azure VM-based (no auto-scale) | Azure PlayFab Matchmaking (auto-scaling) |
| Economy / Inventory DB | Azure SQL General Purpose | Azure Cosmos DB multi-region writes |
| Session Cache | Azure Cache for Redis single-region | Azure Cache for Redis zone-redundant |
| Telemetry | Basic VM metrics | Azure Data Explorer real-time analytics |

### Why the Post Graph Is More Resilient
In the pre-outage graph, Game World Server is reached through a single Azure VM instance provider — one path, no failover. Matchmaking & Session Service has only that same VM plus a single-region Redis cache. The post-outage graph gives Game World Server two independent provider paths (AKS and Azure PlayFab), and Matchmaking & Session Service similarly gains both PlayFab and zone-redundant Redis. Inventory & Cosmetics and Gold & Doubloon Economy each gain two paths through both Cosmos DB multi-region writes and PlayFab, replacing the single Azure SQL path.

---

## 10. Phasmophobia (Kinetic Games)

### Outage / Migration Event
**September–October 2020 — Photon PUN Relay Overload During Viral Surge + 2024 Console Launch**

Phasmophobia launched September 18, 2020 and within days went from hundreds to tens of thousands of concurrent players. The game relied entirely on Photon PUN cloud relay servers and a single-region UK VPS — neither designed for this scale. Photon relay degradation caused lobby creation failures and mid-session disconnections. In 2024, a PS5 and Xbox Series launch required a full multi-platform auth and server orchestration overhaul using Unity Gaming Services.

### Sources
- **Kinetic Games server status thread (Photon):** https://steamcommunity.com/app/739630/discussions/0/2844543519796522749/
- **Photon Engine status (referenced by developer):** https://www.photonengine.com/status
- **GameRevolution server status analysis:** https://www.gamerevolution.com/guides/663267-phasmophobia-server-status-are-the-servers-down-pc
- **Phasmophobia Unity Cloud outage (regional):** https://www.lagofast.com/en/blog/phasmophobia-server-down/
- **Wikipedia (platform info, console launch):** https://en.wikipedia.org/wiki/Phasmophobia_(video_game)

### Infrastructure Changes (Pre → Post)
| Component | Pre-Outage | Post-Outage |
|---|---|---|
| Game Sessions | Photon PUN relay (shared, no dedicated servers) | GCP Cloud Run via Unity Multiplay (dedicated, auto-scaled) |
| Auth | Steam OpenID only | Steam + PSN + Xbox Live via Auth0 B2C (multi-platform) |
| Inventory / Progression | Single-region UK VPS | GCP Firestore via Unity Economy |
| Leaderboard | Single-region UK VPS | GCP Firestore via Unity Cloud Save |
| DDoS Protection | None | Cloudflare Business on public endpoints |

### Why the Post Graph Is More Resilient
In the pre-outage graph, both Lobby & Room Service and Game Session Server are reachable only through Photon PUN — one provider, one path, no failover. Platform Authentication is a single Steam-only path. The post-outage graph gives Lobby & Room Service two independent provider paths (GCP Cloud Run and Cloudflare Business). Platform Authentication gains three separate provider paths (Steam/Valve, PSN, Xbox Live) plus Auth0 B2C as a managed identity layer, meaning auth survives any single platform outage. Equipment & Cosmetics Inventory and Leaderboard & Stats both move from single-region VPS to GCP Firestore, a multi-region serverless provider with a substantially higher resilience score.
