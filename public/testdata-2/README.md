# Game Infrastructure Outage Analysis — 10 Multiplayer Games
## Pre- and Post-Outage Node/Edge CSV Summary

Each game has four CSVs:
- `GameName_pre_nodes.csv` / `GameName_pre_edges.csv` — infrastructure before the outage
- `GameName_post_nodes.csv` / `GameName_post_edges.csv` — infrastructure after remediation

### Schema

**Nodes:** `Node, Type, Service Tier`
- `Type` is either `Functional` (an application feature or service owned by the developer) or `Provider` (a cloud, hosting, or third-party infrastructure provider)
- `Service Tier` is populated only for `Provider` nodes, matching the `Service/Tier` column in `providers.csv`. Left blank for `Functional` nodes and for providers not listed in `providers.csv` (e.g. on-prem DCs, HashiCorp tools, Photon, Steam/Valve)

**Edges:** `Type, From, To`
- `Dependency` — Functional → Functional or Functional → Provider: service A requires service B to function. The dominant edge type; captures how services call, authenticate against, read from, or coordinate with each other
- `Hosted-on` — Functional → Provider: the primary infrastructure a service is deployed on (its compute, database, or CDN home). Used sparingly — one or two per Functional node at most
- No two edges share the same `From` and `To` in the same direction with different types (no Hosted-on/Dependency conflicts). No exact duplicate edges exist.

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
The pre-outage graph had a single MongoDB replica set with no cross-region redundancy; one cache pressure spike on the primary brought down global authentication. The post-outage graph introduces global MongoDB sharding (distributes blast radius), clustered ElastiCache (no single-node SPOF), Fargate multi-AZ compute, and Aurora replacing the slower-failover RDS standby. Kinesis replaces the XMPP stack that saturated Nginx worker threads under load.

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
The pre-outage graph had Consul as an unguarded single point of failure — every service including monitoring depended on one cluster. The post-outage graph distributes Consul across two geographically distinct data centers, so a failure in one cannot take down the entire service mesh. Disabling Consul streaming removes the high-concurrency write contention path, and the BoltDB fix prevents the slow disk exhaustion that exacerbated the failure.

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
The pre-outage graph had a single-threaded, manually scaled server process that could not respond to viral traffic spikes, and the `Game Session Server → Custom Hazel .NET Server` relationship was the sole path for all game traffic (no failover). The post-outage graph moves sessions to Unity Multiplay on GCP Cloud Run with auto-scaling and dedicated server isolation. Cloudflare Business prevents the March 2022-style volumetric attack from reaching origin servers. Auth0 B2C replaces the anonymous session model.

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
| Game Servers | On-prem DCs (Las Vegas, Chile, Brazil) | ECS/EKS Fargate multi-AZ + AWS Global Accelerator |
| Auth / Player Platform | RDS Multi-AZ Standby + EC2 single AZ | Aurora Global Database |
| Backend Services | EC2 single AZ | ECS/EKS Fargate multi-AZ (246 Karpenter clusters) |
| Patch Delivery | CloudFront CDN | CloudFront CDN (unchanged) |

### Why the Post Graph Is More Resilient
On-prem hardware failures had no auto-failover path, causing 90-minute outages. The post-outage graph uses EKS with Karpenter auto-scaling, Aurora Global Database (RTO < 1 min cross-region), and AWS Global Accelerator for sub-35ms latency. The 246-cluster topology means any single cluster failure affects only a geographic slice, not global service.

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
| Anti-Cheat | Easy Anti-Cheat only; no server-side process isolation | EAC + Respawn layered security; sandboxed player-process isolation |
| Auth Cache | ElastiCache single node | ElastiCache cluster mode |
| Account DB | RDS Multi-AZ Standby | RDS Multi-AZ Cluster (2 readable standbys) |
| Public Endpoints | No WAF | Cloudflare Enterprise WAF + DDoS protection |

### Why the Post Graph Is More Resilient
The pre-outage graph had a single anti-cheat layer with no server-side process isolation, allowing one RCE to propagate cheat state directly into the game server's view of the match. The post-outage graph adds sandboxed player-process isolation limiting RCE lateral damage, Respawn's own layered security supplementing EAC, and Cloudflare Enterprise guarding public endpoints from external attack vectors.

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
The pre-outage graph had a self-hosted Mojang auth server with no redundancy, no 2FA, and single-region Azure SQL. The post-outage graph uses Azure AD B2C (Microsoft's enterprise identity platform with global HA), Cosmos DB multi-region active-active writes, and 2FA — directly addressing the account-compromise and single-region failure scenarios that plagued the Mojang era.

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
The pre-outage graph had every service in one physical facility with one database — a single power failure took everything down with no failover path. The post-outage graph distributes critical services across two geographically separate data centers, RDS Multi-AZ provides automatic database failover, and Cloudflare Business protects public web endpoints independently of either DC's health.

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
The pre-outage graph exposed physical server IPs directly through INAP, making them trivially targetable. Routing all traffic through Cloudflare Enterprise hides origin IPs behind anycast scrubbing and WAF, absorbing attack traffic at the global edge before it reaches servers. Moving to a new colo provider removes the shared-infrastructure blast radius. Clustered ElastiCache eliminates session cache as a SPOF.

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
The pre-outage graph had non-redundant single-instance VMs that could not scale to launch demand and had no AZ failover. AKS enables live player migration between containers during updates. PlayFab provides managed server orchestration with auto-scaling. Cosmos DB multi-region writes replace single-region Azure SQL, and zone-redundant Redis eliminates session cache as a single point of failure.

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
The pre-outage graph routed all game sessions through a shared Photon PUN relay with no dedicated server capacity and a single-region VPS as the only backend. The post-outage graph moves sessions to Unity Multiplay on GCP Cloud Run with per-session dedicated server isolation and auto-scaling. GCP Firestore replaces the fragile single-VPS backend with multi-region document storage. Cloudflare Business protects public endpoints, and multi-platform auth via Auth0 B2C replaces the Steam-only identity model.

---

## Schema Reference

### Nodes (`Node, Type, Service Tier`)

| Field | Values |
|---|---|
| `Type` | `Functional` — application feature/service owned by the game developer |
| `Type` | `Provider` — cloud, hosting, or third-party infrastructure provider |
| `Service Tier` | Populated for `Provider` nodes only, matching the `Service/Tier` column in `providers.csv`. Left blank for `Functional` nodes and for providers not in `providers.csv` (e.g. on-prem DCs, HashiCorp tools, Photon, Steam/Valve, Xbox Live, PSN) |

### Edges (`Type, From, To`)

| Field | Values |
|---|---|
| `Type` | `Dependency` — the dominant edge type; Functional → Functional or Functional → Provider, where service A requires service B to operate |
| `Type` | `Hosted-on` — used sparingly; Functional → Provider anchoring a service to its primary compute, database, or CDN provider |

**Integrity guarantees (validated programmatically):**
- Every `From` and `To` value in every edges file exactly matches a `Node` value in the corresponding nodes file
- No two edges share the same `From`/`To` pair with conflicting types (no Hosted-on + Dependency on the same directed pair)
- No exact duplicate edges (same Type, From, To) exist in any file
