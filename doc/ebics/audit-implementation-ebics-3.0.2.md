# Audit d'implémentation EBICS 3.0.2 dans Waarp Gateway

**Date :** 2026-04-01
**Méthode :** Analyse directe du code source confrontée aux spécifications fonctionnelles, techniques et d'architecture logicielle du projet.
**Périmètre :** `pkg/protocols/modules/ebics/`, `pkg/model/ebics_*`, `pkg/admin/rest/ebics_*`, `pkg/cmd/client/ebics_*`, `pkg/backup/ebics_*`, `pkg/ebicsbtfseed/`, `pkg/protocols/modules/ebics/rtn/`, `pkg/protocols/modules/ebics/runtime/`, `pkg/protocols/modules/ebics/stores/`

---

## Table des matières

1. [Synthèse](#1-synthèse)
2. [Protocole EBICS core](#2-protocole-ebics-core)
3. [Ordres administratifs](#3-ordres-administratifs)
4. [Workflows complexes](#4-workflows-complexes)
5. [Écarts critiques identifiés](#5-écarts-critiques-identifiés)
   - [5.1 AMQP 0.9.1 et AMQP 1.0](#51-amqp-091-et-amqp-10--absent)
   - [5.2 RTN (Real-Time Notification)](#52-rtn-real-time-notification--partiel--déconnecté)
   - [5.3 Passe-plat vers application métier](#53-passe-plat-vers-application-métier--partiel)
   - [5.4 Scheduler / ordonnancement automatique](#54-scheduler--ordonnancement-automatique--absent)
   - [5.5 VEU (signature électronique distribuée)](#55-veu-signature-électronique-distribuée--absent-en-tant-que-workflow)
   - [5.6 Tests](#56-tests--lacunaires-sur-plusieurs-fronts)
   - [5.7 Observabilité](#57-observabilité--partielle)
   - [5.8 Rétention / purge](#58-rétention--purge--absent)
6. [Points d'attention crypto / conformité](#6-points-dattention-crypto--conformité)
7. [Tableau récapitulatif des écarts](#7-tableau-récapitulatif-des-écarts)
8. [Conclusion](#8-conclusion)

---

## 1. Synthèse

L'implémentation EBICS de Waarp Gateway est un projet Go structuré autour de deux couches :

- **Bibliothèque externe** `code.waarp.fr/lib/ebics` : protocole H005, parsing XML, crypto, codes retour
- **Gateway** (`pkg/protocols/modules/ebics/`) : persistance, administration, orchestration, observabilité

Le **coeur protocolaire est solide et largement conforme** à EBICS 3.0.2 : ~89 fichiers Go pour le module EBICS, 25 modèles de données, 18+ endpoints REST, un système BTF complet.

Les **écarts majeurs** sont concentrés sur l'**intégration métier et l'automatisation** : AMQP absent, RTN non opérationnel, pas de scheduler, pas de purge automatique.

---

## 2. Protocole EBICS core

| Exigence (spec) | Statut | Détail |
|---|---|---|
| H005 avec validation XSD stricte | **IMPL** | `StrictH005XSDProfile` dans `pkg/protocols/modules/ebics/server.go` |
| H004 (compatibilité EBICS 2.x) | **IMPL** | Supporté, H005 par défaut |
| Modèle 3 phases (init/transfer/receipt) | **IMPL** | Client et serveur |
| BTU/BTD (EBICS 3.0) | **IMPL** | `pkg/protocols/modules/ebics/client_transfer.go`, `order_router.go` |
| FUL/FDL (aliases compatibilité 2.x) | **IMPL** | Normalisation vers BTU/BTD |
| Segmentation configurable | **IMPL** | Défaut 1 Mo, configurable via `maxSegmentSize` |
| Recovery / reprise de transaction | **IMPL** | `pkg/protocols/modules/ebics/client_recovery_store.go` |
| Anti-rejeu (nonce/timestamp) | **IMPL** | `provider_store.go`, TTL 15 min par défaut |
| Codes retour dual-scope (tech/business) | **IMPL** | Séparation stricte dans `EbicsOperation` |
| Crypto A006/X002/E002 | **IMPL** | Via lib-ebics, RSA-2048, SHA-256, AES-CBC |
| Catalogue BTF standard (GLB, FR, DE, AT, CH) | **IMPL** | `pkg/ebicsbtfseed/default_catalogs.json` |
| Résolution de profil payload | **IMPL** | `specific > country > GLB` dans `runtime/payload_resolution.go` |
| Validation de contrat avant émission | **IMPL** | `runtime/contract_validation.go` |
| Retry policy basée sur codes retour | **IMPL** | `runtime/retry_policy.go` |

---

## 3. Ordres administratifs

### Ordres d'initialisation et gestion de clés

| Ordre | Priorité spec | Statut | Fichier |
|---|---|---|---|
| HEV (version protocole) | P1 | **IMPL** | `client_contracts.go` |
| INI (initialisation signature) | P1 | **IMPL** | `client_admin.go` |
| HIA (initialisation auth/chiffrement) | P1 | **IMPL** | `client_admin.go` |
| H3K (certificat signature) | P1 | **IMPL** | `client_admin.go` |
| HPB (clés publiques bancaires) | P1 | **IMPL** | `client_admin.go` |

### Ordres de consultation administrative

| Ordre | Priorité spec | Statut | Fichier |
|---|---|---|---|
| HPD (description protocole hôte) | P2 | **IMPL** | `client_contracts.go` |
| HKD (données clés hôte) | P2 | **IMPL** | `client_contracts.go` |
| HTD (description transfert hôte) | P2 | **IMPL** | `client_contracts.go` |
| HAA (ordres/BTF disponibles) | P2 | **IMPL** | `client_contracts.go` |

### Ordres de rotation de clés

| Ordre | Priorité spec | Statut | Fichier |
|---|---|---|---|
| PUB (publication clé publique) | P2 | **IMPL** | `client_key_rotation.go` |
| HCA (rotation certificat auth) | P2 | **IMPL** | `client_key_rotation.go` |
| HCS (rotation certificat signature) | P2 | **IMPL** | `client_key_rotation.go` |
| HSA (rotation signature) | P2 | **IMPL** | `client_key_rotation.go` |
| SPR (suspension/remplacement) | P2 | **IMPL** | `client_key_rotation.go` |

### Ordres de reporting et signature

| Ordre | Priorité spec | Statut | Fichier |
|---|---|---|---|
| HAC (reporting complet) | P2 | **IMPL** | `client_reporting.go` |
| HVD (historique demande) | P2 | **IMPL** | `client_reporting.go` |
| HVU (historique transfert utilisateur) | P2 | **IMPL** | `client_reporting.go` |
| HVZ (historique transfert signature) | P2 | **IMPL** | `client_reporting.go` |
| HVT (historique transfert) | P2 | **IMPL** | `client_reporting.go` |
| HVE (signature protocolaire) | P2 | **IMPL** | `runtime/signature_state.go` |
| HVS (annulation signature) | P2 | **IMPL** | `runtime/signature_state.go` |

---

## 4. Workflows complexes

| Workflow | Statut | Détail |
|---|---|---|
| Initialisation (INI -> HIA -> H3K -> lettre -> activation) | **IMPL** | `runtime/initialization_runner.go` - machine à états complète avec 7 états |
| Génération lettre EBICS | **IMPL** | Via lib-ebics (`RenderINILetter`, `RenderHIALetter`, `RenderH3KLetter`) |
| Rupture d'automatisation (validation opérateur) | **IMPL** | État `WAITING_LETTER_CONFIRMATION` / `WAITING_BANK_ACTIVATION` |
| Rotation de clés coordonnée multi-clés | **IMPL** | Lifecycle complet : `DRAFT -> MATERIAL_PREPARED -> ORDER_PLANNED -> ORDER_SENT -> WAITING_BANK_CONFIRMATION -> ACTIVATED -> RETIRED` |
| Contract view (projection technique bancaire) | **IMPL** | Modèle + refresh via HPD/HKD/HTD/HAA |
| Payload profiles réutilisables | **IMPL** | `pkg/model/ebics_payload_profile.go` |
| Import / export / updateconf EBICS | **IMPL** | `pkg/backup/ebics_*.go`, migration 0.16.0 |

---

## 5. Écarts critiques identifiés

### 5.1 AMQP 0.9.1 et AMQP 1.0 — ABSENT

**Références spec :**

- Spec fonctionnelle §5.10 : AMQP 0.9.1 (RabbitMQ) et AMQP 1.0 comme modes d'intégration métier minimaux cibles
- Spec technique §7.7 : Connecteurs AMQP avec outbox/consumer workers, supervision, dead-letter
- Architecture §7.9 : AMQP 0.9.1 et 1.0 comme **protocoles Gateway natifs autonomes**, positionnés en **prérequis architectural** (Lot 0) avant intégration EBICS complète

**Constat dans le code :**

- **Aucun code AMQP** dans le projet (`pkg/protocols/modules/amqp091/` et `amqp10/` n'existent pas)
- **Aucune dépendance** AMQP dans `go.mod` (ni `amqp091-go`, ni `go-amqp`)
- Seule la documentation d'architecture existe :
  - `doc/ebics/adr-amqp-et-ebics.md`
  - `doc/ebics/amqp-protocols-backlog.md`
  - `doc/ebics/amqp-protocols-architecture.md`

**Impact :**

- Toute l'intégration métier asynchrone est impossible
- Publication d'événements techniques EBICS vers le SI métier : impossible
- Réception de commandes métier décidées en amont : impossible
- Découplage temporel entre collecte EBICS et consommation applicative : impossible
- RTN vers broker de messages : impossible

**Sévérité : CRITIQUE**

---

### 5.2 RTN (Real-Time Notification) — PARTIEL / DÉCONNECTÉ

**Références spec :**

- Spec fonctionnelle §5.8 : Réception de signal temps réel, validation, journalisation, déduplication, transformation en déclencheur EBICS standard, politique configurable, recovery/replay/quarantaine, observabilité complète
- Spec technique §7.5 : Composante RTN dédiée, intake, idempotence, auto-pull
- Architecture §4.6 : RTN positionné comme source de déclenchement à la frontière de l'architecture

**Constat dans le code :**

| Composant RTN | Statut | Opérationnel ? |
|---|---|---|
| Modèle DB (`ebics_rtn_events`, `ebics_rtn_providers`) | **IMPL** | Oui |
| Tables en base (migration 0.16.0) | **IMPL** | Oui |
| REST API CRUD providers + events + actions retry/quarantine | **IMPL** | Oui |
| CLI (provider add/list/get/update/delete, event list/get/retry/quarantine) | **IMPL** | Oui |
| WSS Provider (`rtn/wss_provider.go`, 284 lignes) | **IMPL** | **Non** |
| Logique d'ingestion (`runtime/rtn_ingestion.go`) avec idempotence SHA-256 | **IMPL** | **Non** |
| Logique auto-pull (`runtime/rtn_autopull.go`) avec plan de pull et corrélation | **IMPL** | **Non** |
| **Service de fond qui instancie et démarre les providers** | **ABSENT** | N/A |
| **Boucle de traitement des événements entrants** | **ABSENT** | N/A |
| **Déclenchement effectif d'un BTD suite à un auto-pull** | **ABSENT** | N/A |
| Publication AMQP des événements RTN | **ABSENT** | N/A |
| Tests unitaires RTN | **ABSENT** | N/A |

**Analyse détaillée :**

Le WSSProvider (`rtn/wss_provider.go`) est un client WebSocket Secure complet : connexion, reconnexion automatique (3s), lecture de messages JSON, normalisation en `RawEvent`, streaming par channel. Mais il n'est **jamais instancié ni démarré** par aucun service.

La logique d'ingestion (`runtime/rtn_ingestion.go`) calcule des clés d'idempotence SHA-256 et gère la déduplication. Mais `IngestRTNEvent()` n'est **jamais appelé** par aucun code de production.

La logique auto-pull (`runtime/rtn_autopull.go`) construit des plans de pull (BTD par défaut) avec corrélation. Mais `BuildAutoPullPlan()` n'est **jamais appelé** pour déclencher un téléchargement réel.

**Verdict :** RTN est une **coquille administrative** — on peut configurer des providers et consulter des events via REST/CLI, mais aucun provider ne se connecte réellement à un endpoint bancaire, aucun événement n'est automatiquement ingéré, et aucun download n'est déclenché. L'architecture RTN est codée à ~60-70% mais fondamentalement **non opérationnelle**.

**Sévérité : MAJEUR**

---

### 5.3 Passe-plat vers application métier — PARTIEL

**Références spec :**

- Spec fonctionnelle §5.9, §5.10 : Connecteurs FS, REST, CLI, AMQP 0.9.1, AMQP 1.0
- Architecture §3.8 : Couche passe-plat pour exposition d'informations techniques EBICS, publication d'événements, réception de commandes métier

**Constat dans le code :**

| Connecteur | Statut |
|---|---|
| Filesystem | **IMPL** (via pipeline Gateway existant) |
| REST API | **IMPL** (endpoints EBICS complets) |
| CLI | **IMPL** (commandes EBICS complètes) |
| AMQP 0.9.1 | **ABSENT** |
| AMQP 1.0 | **ABSENT** |

Sans AMQP, le passe-plat est limité au mode synchrone (REST/CLI) et fichier (FS). Aucun découplage temporel possible entre la Gateway et le SI métier.

**Sévérité : MAJEUR**

---

### 5.4 Scheduler / ordonnancement automatique — ABSENT

**Références spec :**

- Spec fonctionnelle §5.4 : Récupération automatisée de rapports, pulls déclenchés par RTN
- Spec technique §8.2 : `rtnAutoPullPolicy` et `retryPolicy` dans la config client
- Architecture §6.2 : Cas d'usage "client EBICS sortant planifié"

**Constat dans le code :**

- Aucun scheduler, cron, ou mécanisme de déclenchement automatique périodique
- Aucune dépendance de type cron/scheduler dans `go.mod`
- Les transferts sont uniquement déclenchés par :
  - Requête HTTP entrante (serveur)
  - Demande explicite REST/CLI (client)
  - Rule matching du moteur de transfert Gateway

**Sévérité : MOYEN** (peut être partiellement couvert par un ordonnanceur externe)

---

### 5.5 VEU (signature électronique distribuée) — ABSENT en tant que workflow

**Références spec :**

- Spec fonctionnelle §5.3 (ordres de reporting/signature), §6.6 (cas d'usage "signature reçue ou détectée")

**Constat dans le code :**

Les ordres HVE et HVS sont implémentés au niveau protocolaire (`runtime/signature_state.go`), mais il n'existe **aucun workflow de signature distribuée** (VEU / Verteilte Elektronische Unterschrift) qui orchestre :
- La collecte de signatures multiples
- L'exposition vers l'application métier pour décision
- Le suivi de l'état de signature distribué
- La transmission de la décision métier vers la banque

**Sévérité : MOYEN** (la spec positionne les décisions de signature hors Gateway, mais le workflow de transmission reste à implémenter)

---

### 5.6 Tests — LACUNAIRES sur plusieurs fronts

| Zone | Tests | Statut |
|---|---|---|
| Serveur (lifecycle, routing, store, intégration) | Présents | **IMPL** |
| Backup / import / export | Présents | **IMPL** |
| Runtime (contract validation, retry policy) | Présents | **IMPL** |
| Provider store (transaction, segment, nonce) | Présents | **IMPL** |
| RTN (ingestion, auto-pull, WSS provider) | Aucun | **ABSENT** |
| Client EBICS direct (exécution réelle) | Aucun | **ABSENT** |
| REST handlers EBICS (lot B4B) | Partiels | **EN COURS** |
| CLI commands EBICS (lot B4C) | Aucun | **ABSENT** |

**Sévérité : MOYEN** (les tests serveur et runtime sont solides, mais les lacunes RTN et client sont significatives)

---

### 5.7 Observabilité — PARTIELLE

**Références spec :**

- Spec fonctionnelle §8.3 : Logs exploitables par ordre et transaction, corrélation EBICS/Gateway/historique, supervision RTN, supervision unifiée avec autres protocoles
- Spec technique §10 : Traçabilité complète avec HostID/PartnerID/UserID/OrderType/TransactionID, intégration SNMP, métriques

**Constat dans le code :**

- SNMP est implémenté au niveau Gateway global (`pkg/snmp/`), pas spécifiquement pour EBICS
- Les corrélations EBICS existent dans les modèles (`EbicsOperation`, `EbicsTransaction`) mais l'alignement avec les exigences de la spec n'est pas vérifié
- Pas de métriques dédiées EBICS
- Les logs et messages opérateur ne sont pas encore normalisés (étape 5 du plan en attente)

**Sévérité : MOYEN**

---

### 5.8 Rétention / purge — ABSENT

**Références spec :**

- Spec technique §12, Architecture §9 : Purge automatique des nonces expirés, transactions terminées, événements RTN anciens, stores SQL indexés et purgeables

**Constat dans le code :**

- Les nonces ont un TTL (15 min) mais aucun job de nettoyage automatique n'existe
- Pas de politique de purge pour `ebics_transactions` terminées
- Pas de purge pour `ebics_rtn_events` anciens
- Pas de mécanisme de rétention configurable

**Sévérité : MOYEN** (acceptable en développement, problématique en production)

---

## 6. Points d'attention crypto / conformité

### 6.1 Cipher E002 : AES-128-CBC vs AES-256-CBC

La spécification EBICS 3.0.2 (section 14.1) prescrit **AES-128-CBC** pour le chiffrement symétrique des données de commande via le cipher E002. Des références à AES-256-CBC ont été trouvées dans les fichiers de test. Ce point est à vérifier dans `lib/ebics` : si le cipher E002 utilise AES-256-CBC au lieu de AES-128-CBC, cela pourrait poser des problèmes d'interopérabilité avec des serveurs bancaires strictement conformes.

### 6.2 Signature A005 vs A006

L'implémentation utilise A006 (SHA-256) par défaut, ce qui est conforme pour EBICS 3.0.2. A005 (SHA-1) est conservé pour compatibilité descendante. Vérifier qu'A005 n'est jamais utilisé par défaut dans un contexte H005.

### 6.3 VerifyBankKeys désactivable

Le flag `VerifyBankKeys` peut être désactivé (retourne `nil` signer), ce qui bypasse la vérification de signature des réponses bancaires. La spec EBICS 3.0.2 impose cette vérification. Ce flag ne devrait être disponible qu'en environnement de test, avec un avertissement explicite en log.

### 6.4 Clés bancaires : état `imported` vs `validated`

Le flux HPB marque les clés comme `imported` puis nécessite une validation manuelle vers `validated`. Conforme à la spec (vérification par lettre / empreinte SHA-256). Vérifier que le système **refuse les opérations** tant que les clés bancaires sont à l'état `imported`.

### 6.5 H3K comme remplacement INI+HIA

La spec EBICS 3.0.2 prévoit que H3K peut remplacer INI+HIA en une seule étape. Vérifier que cette option est bien supportée dans le workflow d'initialisation.

---

## 7. Tableau récapitulatif des écarts

| Domaine | Spec | Code | Écart | Sévérité |
|---|---|---|---|---|
| Protocole EBICS core (H005, BTU/BTD, crypto) | Oui | **IMPL** | Aucun | - |
| Tous ordres admin (21+ types) | Oui | **IMPL** | Aucun | - |
| Initialisation + lettre EBICS | Oui | **IMPL** | Aucun | - |
| Rotation de clés coordonnée | Oui | **IMPL** | Aucun | - |
| Contract views (HPD/HKD/HTD/HAA) | Oui | **IMPL** | Aucun | - |
| Catalogue BTF standard | Oui | **IMPL** | Aucun | - |
| Import / export / updateconf | Oui | **IMPL** | Aucun | - |
| Retry policy + recovery | Oui | **IMPL** | Aucun | - |
| Payload profiles réutilisables | Oui | **IMPL** | Aucun | - |
| Serveur EBICS (HTTP/TLS) | Oui | **IMPL** | Aucun | - |
| Client EBICS (HTTP/TLS) | Oui | **IMPL** | Aucun | - |
| **AMQP 0.9.1** | Oui (Lot 0) | **ABSENT** | Total | **CRITIQUE** |
| **AMQP 1.0** | Oui (Lot 0) | **ABSENT** | Total | **CRITIQUE** |
| **RTN opérationnel (connexion + auto-pull)** | Oui | **DECONNECTE** | Service manquant | **MAJEUR** |
| **Passe-plat asynchrone vers SI métier** | Oui | **ABSENT** | Dépend AMQP | **MAJEUR** |
| **Scheduler / ordonnancement** | Implicite | **ABSENT** | Total | **MOYEN** |
| **VEU (workflow signature distribuée)** | Oui | **ABSENT** | Workflow manquant | **MOYEN** |
| **Purge / rétention automatique** | Oui | **ABSENT** | Total | **MOYEN** |
| **Observabilité alignée specs** | Oui | **PARTIEL** | Normalisation en attente | **MOYEN** |
| **Tests RTN** | Implicite | **ABSENT** | Total | **MOYEN** |
| **Tests CLI EBICS** | Implicite | **ABSENT** | Total | **MINEUR** |
| **Tests REST handlers EBICS** | Implicite | **EN COURS** | Partiel | **MINEUR** |

---

## 8. Conclusion

### Ce qui est solide

Le **coeur protocolaire EBICS est conforme à la spécification 3.0.2** et de qualité production :

- Les 21+ types d'ordres sont implémentés (admin, payload, reporting, signature)
- Le modèle 3 phases (initialisation, transfert segmenté, quittance) est complet
- La crypto (A006/X002/E002) est correctement déléguée à lib-ebics
- Les workflows complexes (initialisation avec lettre, rotation de clés coordonnée) sont opérationnels
- Le catalogue BTF standard couvre 5 périmètres géographiques (GLB, FR, DE, AT, CH)
- La gestion des codes retour dual-scope (technique/business) est rigoureuse
- La persistance (25 modèles, migration, import/export) est complète
- L'API REST et le CLI couvrent toutes les familles d'objets EBICS

### Ce qui manque

Les écarts sont concentrés sur **l'intégration métier et l'automatisation** :

1. **AMQP (0.9.1 + 1.0)** : Identifié comme prérequis architectural (Lot 0) dans les specs, pas une seule ligne de code n'existe. Bloque tout le passe-plat asynchrone vers le SI métier.

2. **RTN non opérationnel** : Le code est écrit à ~60-70% (WSS provider, ingestion, auto-pull) mais les briques ne sont pas connectées entre elles. Le WSSProvider ne démarre jamais, les événements ne sont jamais ingérés automatiquement, l'auto-pull ne déclenche jamais de BTD. Actuellement une façade administrative sans exécution réelle.

3. **Pas de scheduler** : Aucun mécanisme de déclenchement automatique périodique (rapports, refresh contrats, retry programmé).

4. **Pas de purge** : Les nonces, transactions terminées et événements RTN s'accumulent sans limite.

### Priorisation recommandée

| Priorité | Action | Justification |
|---|---|---|
| P0 | Connecter le RTN (service de fond + boucle d'ingestion + déclenchement auto-pull) | Le code est à 60-70%, il faut principalement un service orchestrateur |
| P1 | Implémenter AMQP 0.9.1 puis 1.0 comme protocoles Gateway natifs | Prérequis architectural pour l'intégration métier asynchrone |
| P2 | Ajouter un mécanisme de purge/rétention automatique | Nécessaire avant mise en production |
| P2 | Compléter les tests (RTN, CLI, client direct) | Couverture critique manquante |
| P3 | Normaliser l'observabilité (logs, corrélations, messages opérateur) | Alignement avec specs fonctionnelles |
| P3 | Implémenter le workflow VEU / signature distribuée | Dépend de la stratégie passe-plat métier |
| P3 | Ajouter un scheduler intégré ou documenter l'utilisation d'un ordonnanceur externe | Peut être couvert par outillage existant |
