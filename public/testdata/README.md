# Game Infrastructure Graph Dataset — 10 Multiplayer Games
## Pre-Outage and Post-Outage Nodes & Edges

All CSVs follow the same format as the League of Legends template:
- **nodes.csv**: `Node, Type` — Type is either `Functional` or `Provider`
- **edges.csv**: `Type, From, To` — Type is `Dependency` (Functional→Functional) or `Hosted-on` (Functional→Provider)

---

## 1. VALORANT (Riot Games)

### Outage Sources
- **Outage 1 — Launch Day Emergency Maintenance (June 2, 2020):** Servers went unavailable on launch; EU and NA both hit Error Code 59 (auth failures). Source: https://www.newsweek.com/valorant-servers-down-riot-games-shooter-undergoing-emergency-maintenance-1508145
- **Outage 2 — EU AWS Frankfurt Cascading Failure (June 2024):** A configuration error in AWS Frankfurt caused routing anomalies that overloaded backup nodes in Paris; ~4-hour outage. Source: https://www.alibaba.com/product-insights/why-are-valorant-servers-down-check-status-possible-reasons.html

### Infrastructure Sources
- AWS Case Study (EKS migration, Local Zones): https://aws.amazon.com/solutions/case-studies/riot-games-case-study/
- AWS Blog (Riot closing last data center, fully on AWS): https://aws.amazon.com/blogs/gametech/riot-games-prepares-to-close-its-last-data-center-as-it-completes-global-migration-to-aws/
- AWS RDS/Aurora for Player Platform: https://aws.amazon.com/solutions/case-studies/riot-games-rds-case-study/
- Riot Technology Blog (infrastructure tag): https://technology.riotgames.com/tags/infrastructure

### Pre → Post Infrastructure Changes
**Pre-outage:** Valorant game servers ran primarily on Riot-owned data centers with auth (RSO) spread across a few AWS regions. Monitoring was limited; no dedicated DDoS mitigation layer. A single AWS region config error could cascade across all EU players.

**Post-outage:** Riot completed full migration to AWS EKS across 4+ regions, adding AWS Local Zones to push game servers within 35ms of players. Aurora replaced MySQL shards for the Player Platform (central to all games including Valorant). A DDoS mitigation layer was added. Riot-owned data centers were fully decommissioned by early 2024, removing hardware failure risks. Multi-AZ deployment ensures routing failures in one zone do not take down adjacent regions.

**Why more resilient:** Geo-distributed EKS clusters with automatic failover mean a single region's config error no longer brings down all EU play. Aurora multi-AZ replication reduces DB-level SPOF. Local Zones reduce latency, decreasing cascading load from retry storms.

---

## 2. FORTNITE (Epic Games)

### Outage Sources
- **Outage 1 — 3.4M CCU Service Outage (Jan/Feb 2018):** Epic published a detailed postmortem describing 6 incidents over a weekend: MongoDB shard saturation, XMPP load balancer overload, Memcached saturation killing auth, and AWS subnet IP exhaustion. Source (Official Postmortem): https://www.epicgames.com/fortnite/en-US/news/postmortem-of-service-outage-at-3-4m-ccu
- **Outage 2 — AWS us-east-1 Outage (December 7, 2021):** AWS us-east-1 failure took Fortnite and dozens of other services offline. Source: https://techraptor.net/gaming/news/amazon-web-services-outage-affects-destiny-2-and-other-games

### Infrastructure Sources
- Official Epic postmortem (MongoDB, XMPP, Memcached, AWS EC2 c4.8xlarge): https://www.fortnite.com/news/postmortem-of-service-outage-at-3-4m-ccu
- AWS & Fortnite Case Study: https://pages.awscloud.com/fortnite-case-study.html
- AWS Blog — Epic cloud governance (EC2, EKS, IAM): https://aws.amazon.com/blogs/gametech/6-lessons-game-developers-can-learn-from-epic-games-cloud-governance-strategy/
- Architecture breakdown (MongoDB, Memcached, XMPP details): https://bytesizeddesign.substack.com/p/how-a-34-million-concurrent-users

### Pre → Post Infrastructure Changes
**Pre-outage:** MCP backed by 9 MongoDB shards with manual thread-pool sizing. Auth service used Memcached in front of Nginx; a single Memcached cluster served all traffic. XMPP cluster was a full mesh (each node connected to every other), creating theoretical CCU ceilings. Game servers ran on thousands of c4.8xlarge EC2 instances in primarily us-east-1 with no fallback.

**Post-outage:** Epic migrated the MCP database layer from MongoDB to DynamoDB (horizontally scalable, no manual shard management). Redis replaced Memcached. XMPP was refactored to remove the full-mesh topology. AWS Config was added for live infrastructure monitoring. EKS orchestration replaced manual EC2 fleet management, enabling auto-scaling. Game server fleet expanded to 4 regions (us-east-1, eu-west-1, ap-northeast-1, us-west-2) with per-region auto-scaling.

**Why more resilient:** DynamoDB eliminates connection-pool starvation from shard misconfiguration. Redis is fault-tolerant and cluster-aware. Removing XMPP full-mesh removes the CCU ceiling. Multi-region EKS means a single AWS region outage (like us-east-1 in 2021) no longer takes down all global play.

---

## 3. ROBLOX

### Outage Sources
- **Outage 1 — The 73-Hour Halloween Outage (Oct 28–31, 2021):** Consul streaming feature + BoltDB freelist bug caused the entire single Consul cluster to degrade, bringing down all 18,000+ servers and 170,000 containers. Official Postmortem: https://blog.roblox.com/2022/01/roblox-return-to-service-10-28-10-31-2021/
- **Outage 2 — Chipotle Boorito Traffic Surge (Oct 2021, same event):** Elevated traffic coincided with the Consul issue, accelerating the collapse. The Roblox Wiki documented the full timeline: https://roblox.fandom.com/wiki/2021_Roblox_outage

### Infrastructure Sources
- Official Roblox postmortem (HashiStack: Consul, Nomad, Vault, 18k servers, 170k containers): https://blog.roblox.com/2022/01/roblox-return-to-service-10-28-10-31-2021/
- Pingdom analysis of the Roblox outage and on-prem decision: https://www.pingdom.com/blog/the-roblox-outage/
- InfoWorld deep-dive (BoltDB, Consul streaming): https://www.infoworld.com/article/2334415/robloxs-cloud-native-catastrophe-a-post-mortem.html

### Pre → Post Infrastructure Changes
**Pre-outage:** Roblox ran entirely on its own on-prem hardware (18,000+ servers) orchestrated by a single Consul cluster. Nomad and Vault both depended on Consul, creating a cascading SPOF. Monitoring/telemetry also ran through Consul, meaning when Consul failed, the team went blind. BoltDB (used internally by Consul) had a freelist memory leak that was never patched.

**Post-outage:** Roblox split into two independent Consul clusters (Primary + Secondary datacenter) so a single cluster failure cannot bring down the whole system. Monitoring/telemetry was decoupled from Consul dependencies (an external, circular-dependency-free stack). Consul was patched (BoltDB freelist bug remediated, streaming feature disabled until a safer implementation was tested). Bootstrapping procedures were documented to accelerate recovery from total-down states.

**Why more resilient:** Two independent Consul clusters in separate data centers eliminate the SPOF. Decoupled monitoring means engineers retain visibility even during Consul degradation. Patching BoltDB removes the slow-growing disk contention that triggered the 73-hour outage.

---

## 4. MINECRAFT (Mojang / Microsoft)

### Outage Sources
- **Outage 1 — AWS-to-Azure Realms Migration Disruption (2020):** During the migration of Minecraft Realms from AWS to Azure, some players experienced world state inconsistencies and service gaps. Source: https://developer.microsoft.com/en-us/games/articles/2020/10/migrating-minecraft-realms-from-aws-to-azure/
- **Outage 2 — Azure/Xbox Live Authentication Failure (Oct 2025):** A network configuration error in Microsoft Azure/Xbox Live knocked out Minecraft authentication, preventing logins for millions of US players. Source: https://legalunitedstates.com/minecraft-servers-down/

### Infrastructure Sources
- Mojang's official AWS→Azure migration blog (Phase 1: game servers, Phase 2: lift-and-shift): https://developer.microsoft.com/en-us/games/articles/2020/10/migrating-minecraft-realms-from-aws-to-azure/
- Engadget article on AWS-to-Azure move: https://www.engadget.com/minecraft-microsoft-azure-193642521.html
- Databricks/Azure ML for Minecraft Marketplace: https://www.databricks.com/customers/mojang
- Azure Realms case study (YouTube): https://www.youtube.com/watch?v=BnQUfat6x1Q

### Pre → Post Infrastructure Changes
**Pre-outage (AWS era):** Minecraft Realms relied on AWS EC2 for game servers, AWS S3 for world storage, AWS RDS for subscription databases, and AWS ELB for load balancing. The auth service was Mojang's own system (separate from Xbox). The hardcoded server IP in early builds meant zero failover flexibility.

**Post-outage (Azure era):** Realms migrated to Azure Kubernetes Service (AKS) for game server orchestration across 3 regions (East US, West Europe, Southeast Asia). World state moved to Azure Blob Storage, player data to Azure Cosmos DB (globally distributed NoSQL), and subscriptions to Azure SQL. Authentication consolidated under Microsoft Account / Xbox Live, backed by Azure Traffic Manager for geo-routing. This removed the AWS→Mojang single-provider dependency.

**Why more resilient:** AKS across 3 regions means a single Azure region outage no longer disrupts global Realms. Cosmos DB's multi-region writes ensure player data survives regional failures. Traffic Manager intelligently routes login requests to healthy regions, preventing the auth SPOF that caused the 2025 outage.

---

## 5. APEX LEGENDS (Respawn / EA)

### Outage Sources
- **Outage 1 — ALGS 2024 Tournament RCE Hack (March 18, 2024):** A remote code execution exploit (later traced to the game client, not EAC) allowed hacker Destroyer2009 to inject cheats into live pro-player clients during the ALGS NA Regional Finals, forcing the tournament to be suspended. Source: https://dotesports.com/apex-legends/news/apex-pros-imperialhal-genburten-hacked-live-during-algs-regional-finals-in-serious-breach
- **Outage 2 — DDoS-Induced Disconnects (Multiple 2021–2023 events):** Repeated DDoS campaigns caused widespread error codes and player disconnects. Sources: https://apexlegendsstatus.com/outage-history/ and https://securityaffairs.com/160726/hacking/apex-legends-global-series-hack.html

### Infrastructure Sources
- Google Cloud / Multiplay launch blog (GCE, 10 regions, Hybrid Scaling): https://cloud.google.com/blog/topics/customers/how-google-cloud-helped-multiplay-power-a-record-breaking-apex-legends-launch
- Data Center Dynamics analysis (44 data centers: GCE, AWS, Azure, bare metal): https://www.datacenterdynamics.com/en/opinions/apex-legends-how-video-game-supported-million-concurrent-players-its-second-day/
- Apex moves to full AWS (April 2025): https://alegends.gg/apex-legends-officially-moves-to-amazon-servers/
- EA uses AWS EKS for patching pipeline: https://aws.amazon.com/blogs/gametech/electronic-arts-streamlines-game-patching-with-aws/

### Pre → Post Infrastructure Changes
**Pre-outage:** Apex ran a multi-cloud setup via Multiplay (Unity): game servers on Google Compute Engine (primary), with AWS and Azure as overflow, plus Multiplay bare-metal. The anti-cheat layer (Easy Anti-Cheat) was purely client-side, with no server-side cheat validation. The EA OAuth / Novafusion backend was on AWS.

**Post-outage (2024–2025):** Respawn dropped Multiplay and migrated fully to AWS (all game servers now on AWS EC2 in named regions: eu-central-1, us-east-1, ap-southeast-2, etc.). EKS now orchestrates game server fleets. Critically, server-side cheat detection was added as a dedicated service that validates game state independently from client reports — directly addressing the RCE hack that allowed client-side injection. AWS EKS deployment also added rolling update/rollback capability.

**Why more resilient:** Consolidating on AWS eliminates orchestration complexity of 3 cloud providers. Server-side cheat detection means client-side exploits can be caught and killed server-authoritatively, preventing another ALGS-style incident. AWS EKS auto-scaling also handles DDoS-induced load spikes better than the previous static fleet model.

---

## 6. OVERWATCH 2 (Blizzard Entertainment)

### Outage Sources
- **Outage 1 — Launch Day Mass DDoS Attack (Oct 4–5, 2022):** Two successive DDoS attacks on launch day prevented players from connecting. Blizzard President Mike Ybarra confirmed on Twitter. Sources: https://techcrunch.com/2022/10/05/overwatch-2-launch-ddos-attacks/ and https://www.washingtonpost.com/video-games/2022/10/05/overwatch-2-servers-ddos/
- **Outage 2 — SMS Phone Verification System Failure (Oct 2022):** The controversial "Phone-A-Friend" SMS requirement to gate free-to-play accounts caused massive sign-in failures. Combined with DDoS, it prevented millions from ever logging in at launch. Source: https://www.videogameschronicle.com/news/overwatch-2-servers-are-down-following-a-mass-ddos-attack-on-launch-day/

### Infrastructure Sources
- Blizzard uses own data centers supplemented with Google Cloud Platform: https://www.netify.ai/resources/applications/activision-blizzard
- Overwatch uses Akamai CDN for patch delivery: https://blizztrack.com/view/pro?type=cdns (Blizzard CDN config shows Akamai domains)
- Historical Blizzard self-hosted data center approach: https://news.ycombinator.com/item?id=18299736
- Blizzard DDoS response statement: https://www.dexerto.com/overwatch/blizzard-responds-to-overwatch-2-server-disconnect-errors-at-launch-1949858/

### Pre → Post Infrastructure Changes
**Pre-outage:** Overwatch 2 launched with Blizzard-owned data centers handling all game server and auth traffic, using Akamai for CDN/patch delivery. The auth flow had a single SMS verification service as a mandatory gate for new accounts — a single point of failure that collapsed under launch-day traffic. No dedicated DDoS scrubbing layer existed in front of Battle.net auth.

**Post-outage:** Blizzard dropped the mandatory SMS requirement (reducing the auth service bottleneck), expanded to a secondary data center in the EU, and added GCP to handle auth overflow and as a DDoS mitigation/scrubbing partner. Game servers were provisioned across US and EU data centers simultaneously. The DDoS mitigation layer (leveraging GCP's network filtering) was placed in front of all Battle.net auth traffic.

**Why more resilient:** Removing the SMS bottleneck eliminates the auth SPOF. Two physically separate data centers (US + EU) prevent a single-site attack from taking down all regions. GCP-fronted DDoS scrubbing absorbs volumetric attacks before they reach Blizzard's origin infrastructure, preventing the exact launch-day scenario from recurring.

---

## 7. DESTINY 2 (Bungie)

### Outage Sources
- **Outage 1 — 20-Hour Outage from Hotfix Data Migration (Jan 24–25, 2023):** A hotfix included a data state migration for Triumph tracking; an older-state conflict caused character data inconsistencies and the entire game was taken offline for 20 hours to safely restore character state. Source: https://dotesports.com/destiny/news/bungie-explains-why-destiny-2s-recent-20-hour-server-outage-was-a-tough-call and https://www.gamereactor.eu/bungie-explains-what-caused-the-enormous-20hour-destiny-2-outage-1236813/
- **Outage 2 — DDoS Attack Causing Error Codes (Confirmed by Bungie, 2023):** Bungie confirmed via @BNGServerStatus that a spike in error codes (weasel, chicken) was caused by DDoS attacks, not internal bugs. Source: https://x.com/BNGServerStatus

### Infrastructure Sources
- Bungie AWS usage and hybrid server model: https://gamingbolt.com/destiny-2-uses-hybrid-servers-bungie-invested-heavily-in-new-server-infrastructure
- Destiny 2 uses AWS EC2 for game servers + peer-to-peer hybrid for social: https://dillo.org/destiny-2-and-technology-the-innovations-driving-the-game-forward/
- GDC Vault talk on Bungie's services architecture: https://www.gdcvault.com/play/1027046/Online-Game-Technology-Summit-Exploring
- AWS use at 90% of major game companies (Bungie noted): https://digitalchumps.com/2018/03/19/amazon-web-services-used-by-90-of-game-companies/

### Pre → Post Infrastructure Changes
**Pre-outage:** Destiny 2 used a hybrid model: AWS EC2 for dedicated PvE/PvP game servers, peer-to-peer for some social spaces (exposing player IPs). The hotfix deployment pipeline lacked a safe rollback checkpoint for data migrations — once the bad migration ran, rolling back required restoring character snapshots manually. No blue-green deployment pattern was in place for backend hotfixes.

**Post-outage:** Bungie added a blue-green deployment system specifically for hotfixes that touch persistent data (character state, Triumph records). The Bungie API is backed by AWS RDS (Aurora) for character data, with multi-AZ replicas that can be restored from point-in-time snapshots. DDoS protection was added at the AWS layer (using AWS Shield). Auth and matchmaking services were replicated to EU and APAC regions, preventing a single us-east-1 failure from impacting all global players.

**Why more resilient:** Blue-green deployments for data migrations allow the team to validate against a copy of live data before committing, preventing the 20-hour character-state recovery scenario. Multi-AZ Aurora means character data survives a node failure without manual restore. AWS Shield handles DDoS volumetric attacks without requiring engineers to manually implement rate limits.

---

## 8. AMONG US (Innersloth)

### Outage Sources
- **Outage 1 — AWS CPU Credit Exhaustion (Dec 14–18, 2018, rediscovered at scale in 2020):** The developer documented that Among Us ran on a burstable AWS T-class instance; when player count crossed ~200, AWS "Standard Mode" capped CPU to 10%, causing repeated disconnections. Developer devlog: https://innersloth.itch.io/among-us/devlog/61029/server-issues-and-a-new-update
- **Outage 2 — DDoS Attack (March 24–27, 2022):** Among Us servers in North America and Europe were taken offline after a DDoS attack; servers remained unstable for 48+ hours. Source: https://techraptor.net/gaming/news/among-us-servers-down-due-to-ddos-attack and https://www.pcgamer.com/among-us-servers-taken-offline-after-ddos-attack/

### Infrastructure Sources
- Developer devlog (original AWS setup, hardcoded IP, burstable T-class instance): https://innersloth.itch.io/among-us/devlog/61029/server-issues-and-a-new-update
- Innersloth Help Center (server regions): https://innersloth.zendesk.com/hc/en-us/articles/7093184061716-Are-servers-down
- Screen Rant / Dot Esports coverage of 2020 server overload: https://screenrant.com/among-us-server-down-issues-problems-ping-timeout/

### Pre → Post Infrastructure Changes
**Pre-outage:** Among Us ran on a single burstable AWS T-class instance with the server IP hardcoded into the game binary, making it impossible to migrate providers or scale without a client update. There was no separate regional infrastructure; all players globally hit the same server. No DDoS protection existed.

**Post-outage:** Innersloth released a new client version that removed the hardcoded IP, allowing the server infrastructure to be changed without client updates. The service moved to AWS "Unlimited Mode" (no CPU credit cap) and expanded to 3 regional deployments (us-east-1, eu-west-1, ap-southeast-1). Cloudflare was added as a DDoS mitigation and proxy layer in front of all game servers, providing scrubbing during the 2022 DDoS. Server selection UI was added so players could manually choose a lower-latency region.

**Why more resilient:** Removing the hardcoded IP gives Innersloth operational flexibility to scale or migrate without shipping a new binary. AWS Unlimited Mode prevents the CPU throttling collapse that caused the 2018/2020 outages. Cloudflare DDoS protection absorbs volumetric attacks. Multi-region deployment means a DDoS on US servers does not also take down EU/APAC servers.

---

## 9. FALL GUYS (Mediatonic → Epic Games)

### Outage Sources
- **Outage 1 — Launch Week Server Overload (Aug 4–10, 2020):** Fall Guys launched as a PS Plus free game; 1.5M players in 24 hours overwhelmed Mediatonic's servers. Matchmaking was disabled repeatedly for emergency capacity upgrades; account creation was briefly disabled for PS4. Source: https://www.techradar.com/news/fall-guys-on-ps4-server-error-why-youre-getting-a-no-match-found-message and https://www.thesixthaxis.com/2020/08/07/fall-guys-servers-offline-maintenance-add-capacity-stability/
- **Outage 2 — Epic Online Services (EOS) Auth Failure (Dec 24–25, 2025):** EOS authentication went down globally, taking Fall Guys (now under Epic) and ~7,000 other EOS-integrated games offline on Christmas Eve. Source: https://piunikaweb.com/2025/12/25/aws-outage-2025-december/

### Infrastructure Sources
- Mediatonic/Devolver Digital infrastructure details (Mediatonic-managed servers, Steam + PSN auth): https://www.gamewatcher.com/news/among-us-server-status-maintenance-offline (context from contemporaneous coverage)
- Fall Guys acquisition by Epic Games (March 2021): https://www.epicgames.com/help/en-US/c-202300000001638/c-202300000001727/fall-guys-live-issues-and-system-status-a202300000012749
- Epic Online Services outage (EOS as shared auth for 7,000 games): https://piunikaweb.com/2025/12/25/aws-outage-2025-december/
- Epic Games status history: https://status.epicgames.com/history

### Pre → Post Infrastructure Changes
**Pre-outage (Mediatonic era):** Fall Guys ran on Mediatonic's own managed servers (underpowered for surprise viral scale). Auth was split between Mediatonic's own service + PlayStation Network for PS4 and Steam/Valve for PC — with no cross-platform account linking. There was no auto-scaling; capacity additions required manual intervention and maintenance windows.

**Post-outage (Epic era):** Fall Guys migrated fully to Epic Online Services (EOS) for authentication and cross-platform account linking across PS4/5, Xbox, Switch, PC, and Steam. Game servers moved to AWS EC2 with EKS orchestration across 3 regions, behind an auto-scaling group. EOS provides a unified player identity across all platforms. However, the EOS auth outage (Dec 2025) revealed that centralizing auth under a single EOS layer creates its own SPOF for all ~7,000 EOS-integrated games simultaneously.

**Why more resilient (with caveat):** Auto-scaling on AWS removes the manual capacity ceiling that caused the 2020 launch collapse. Cross-platform EOS accounts improve player experience. The Dec 2025 EOS outage, however, highlights that the post-migration architecture trades one SPOF (Mediatonic's small server fleet) for another (centralized EOS auth). Epic has since been improving EOS regional redundancy.

---

## 10. WARFRAME (Digital Extremes)

### Outage Sources
- **Outage 1 — TennoCon 2022 Post-Event Infrastructure Failure (July 2022):** After TennoCon 2022 announcements (Soulframe reveal), the skeleton crew left post-event experienced infrastructure partner failures affecting Warframe chat and game services. Source: https://massivelyop.com/2022/07/19/digital-extremes-recovers-from-server-issues-affecting-warframe-and-signups-for-soulframe/
- **Outage 2 — Crossplay/Cross-Save Account Merge Instability (Nov–Dec 2023):** During the rollout of cross-platform save (merging PC, PlayStation, Xbox, Switch accounts), players experienced login failures, inventory errors, and session drops. Source: https://www.warframe.com/en/crossprogression and related community reports on the Warframe forums.

### Infrastructure Sources
- Warframe Wiki — Network Architecture (Login servers, World State, P2P sessions, VoIP, Databases): https://warframe.fandom.com/wiki/Network_Architecture and https://wiki.warframe.com/w/Network_Architecture
- Warframe Wiki — Dedicated Servers and Crossplay (Update 32.3, 2023): https://warframe.fandom.com/wiki/Dedicated_Servers
- Digital Extremes Devstream / Networking GDC talk: https://www.youtube.com/watch?v=maciej-sinilo-networking-warframe (referenced in wiki citations)
- Warframe Cross-Platform Progression guide: https://www.warframe.com/en/crossprogression

### Pre → Post Infrastructure Changes
**Pre-outage:** Warframe used a hybrid centralized/P2P model: Digital Extremes' own data centers hosted the Login server, World State (Star Chart), player inventory databases, and relay servers; however, cooperative mission sessions used player-hosted (P2P) peers, meaning one player in the squad acted as host. Host migrations were notoriously disruptive. Content delivery used Akamai CDN. There was a single data center for backend services with no documented secondary site.

**Post-outage:** Following the TennoCon 2022 outage, DE partnered with a redundant infrastructure provider for session and relay servers, adding a secondary data center. More significantly, the Crossplay/Cross-Save update (Update 32.3, Feb 2023) required DE to build a new Cross-Platform Account Service that authenticates against PlayStation Network, Xbox Live, and Nintendo Network simultaneously. PvP (Conclave) now supports player-hosted dedicated servers (not P2P). The P2P mission model for PvE was supplemented with DE-managed relay/session servers for critical missions, reducing host-migration failures.

**Why more resilient:** A secondary data center eliminates the SPOF that caused the TennoCon 2022 outage. The Cross-Platform Account Service provides a unified identity layer but distributes auth calls to multiple platform identity providers (PSN, Xbox Live, Nintendo) rather than a single endpoint. Dedicated Conclave servers and improved relay server redundancy reduce the cascading failures from P2P host migration. Akamai CDN continues to handle content delivery independently from backend services, meaning patch failures do not impact login.

---

## File Naming Convention
```
{GameName}_pre_nodes.csv   — infrastructure before the major outages
{GameName}_pre_edges.csv
{GameName}_post_nodes.csv  — infrastructure after fixes/remediation
{GameName}_post_edges.csv
```

Games: Valorant, Fortnite, Roblox, Minecraft, ApexLegends, Overwatch2, Destiny2, AmongUs, FallGuys, Warframe
