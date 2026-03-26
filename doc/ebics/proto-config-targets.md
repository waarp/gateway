# Cibles `ProtoConfig` pour `amqp091`, `amqp10` et `ebics`

## 1. Objet

Ce document fixe une cible de design pour:

- les `ProtoConfig` des nouveaux protocoles `amqp091` et `amqp10`;
- les `ProtoConfig` cibles d'EBICS;
- leur representation dans les fichiers JSON / YAML utilises par
  `export/import` et `updateconf`.

L'objectif n'est pas de figer tous les champs definitifs, mais de definir une
forme suffisamment stable pour les spikes et la validation d'architecture.

## 2. Principes communs

Les `ProtoConfig` doivent respecter les regles suivantes:

- etre purement serialisables en JSON / YAML;
- ne contenir aucun secret obligatoire en clair si une `Credential` Gateway
  peut porter l'information;
- separer les champs de connectivite des politiques d'execution;
- rester comprehensibles dans les sauvegardes et archives `updateconf`.

## 3. `amqp091`

## 3.1 `Client.ProtoConfig`

```json
{
  "uri": "amqps://broker.example.net:5671",
  "virtualHost": "/",
  "exchange": "gateway.out",
  "exchangeType": "topic",
  "routingKeyTemplate": "gateway.${eventType}",
  "mandatory": true,
  "persistentMessages": true,
  "publisherConfirms": true,
  "heartbeatSeconds": 30,
  "connectionName": "gateway-amqp091-out",
  "tlsProfile": "default",
  "retryPolicy": {
    "initialDelaySeconds": 5,
    "maxDelaySeconds": 300,
    "maxAttempts": 0
  }
}
```

## 3.2 `RemoteAgent.ProtoConfig`

```json
{
  "virtualHost": "/",
  "exchange": "gateway.in",
  "queue": "gateway.in.queue",
  "queueDurable": true,
  "bindingKeys": [
    "gateway.command.*",
    "gateway.reply.*"
  ],
  "consumerTag": "gateway-amqp091-in",
  "prefetchCount": 20,
  "autoAck": false,
  "tlsProfile": "default"
}
```

## 4. `amqp10`

## 4.1 `Client.ProtoConfig`

```json
{
  "endpoint": "amqps://broker.example.net:5671",
  "targetAddress": "gateway/out",
  "senderLinkName": "gateway-amqp10-out",
  "settlementMode": "mixed",
  "durable": true,
  "idleTimeoutSeconds": 60,
  "maxInFlight": 100,
  "tlsProfile": "default",
  "retryPolicy": {
    "initialDelaySeconds": 5,
    "maxDelaySeconds": 300,
    "maxAttempts": 0
  }
}
```

## 4.2 `RemoteAgent.ProtoConfig`

```json
{
  "sourceAddress": "gateway/in",
  "receiverLinkName": "gateway-amqp10-in",
  "credit": 50,
  "settlementMode": "peek-lock",
  "idleTimeoutSeconds": 60,
  "tlsProfile": "default"
}
```

## 5. `ebics`

## 5.1 `Client.ProtoConfig`

```json
{
  "hostId": "BANKHOST01",
  "partnerId": "PARTNER01",
  "userId": "USER01",
  "productName": "Waarp Gateway",
  "productVersion": "dev",
  "ebicsVersion": "H005",
  "urlPath": "/ebicsweb",
  "requestTimeoutSeconds": 120,
  "segmentSize": 1048576,
  "contractRefreshPolicy": {
    "startup": true,
    "periodicHours": 24,
    "onErrorCodes": [
      "091116",
      "091117"
    ]
  },
  "reportingPolicy": {
    "enableAutoDownload": true,
    "orderTypes": [
      "HAC",
      "VMK",
      "STA"
    ],
    "useRtn": true
  },
  "businessIntegrationProfile": "amqp10-default",
  "payloadSubmissionPolicy": "profile-preferred",
  "defaultPayloadProfiles": {
    "BTU": "btu-default",
    "BTD": "btd-default",
    "FUL": "ful-default",
    "FDL": "fdl-default"
  },
  "defaultTransferRuleSend": "ebics-send-default",
  "defaultTransferRuleReceive": "ebics-receive-default"
}
```

## 5.2 `RemoteAgent.ProtoConfig`

```json
{
  "hostId": "BANKHOST01",
  "urlPath": "/ebicsweb",
  "tlsProfile": "default",
  "bankPublicKeyProfile": "bank-main",
  "supportedVersions": [
    "H005"
  ]
}
```

## 6. Position sur les secrets

Les champs suivants ne doivent pas etre la forme privilegiee de stockage dans
`ProtoConfig`:

- mots de passe;
- clefs privees;
- certificats sensibles;
- secrets SASL si on ajoute d'autres modes d'authentification.

Ils doivent etre portes preferentiellement par:

- `Credential`;
- ou un mecanisme de coffre/delegation de secret si Gateway en dispose.

## 7. Position sur `Rule` pour EBICS

Les champs:

- `defaultTransferRuleSend`
- `defaultTransferRuleReceive`

sont volontaires.

Ils actent la cible d'integration suivante:

- les ordres EBICS administratifs ne referencent pas de `Rule`;
- les flux fichier EBICS, eux, peuvent etre projetees vers des `Transfer`
  Gateway et utilisent alors une `Rule` technique.

Autrement dit:

- `Rule` reste dans la boucle;
- mais seulement pour la projection fichier de certains usages EBICS.

Complement ergonomique:

- les ordres payload EBICS doivent pouvoir referencer des profils payload
  dedies;
- ces profils portent la semantique `serviceName/serviceOption/scope/msgName`;
- `Rule` reste concentree sur la technique Gateway.
- la politique produit de soumission payload doit etre parametrable via
  `payloadSubmissionPolicy`.

## 8. Impact `updateconf`

Les fichiers JSON / YAML transportes par `export/import` et `updateconf`
doivent pouvoir contenir sans adaptation particuliere:

- des `Client` de protocole `amqp091`, `amqp10`, `ebics`;
- des `RemoteAgent` de protocole `amqp091`, `amqp10`, `ebics`;
- des `ProtoConfig` heterogenes mais stables;
- des `Rule` techniques dediees aux flux fichier EBICS si elles existent.

Les exemples de configuration doivent inclure au minimum:

- un jeu `amqp091`;
- un jeu `amqp10`;
- un jeu `ebics`;
- un cas EBICS avec regles techniques minimales.

## 9. Questions de spike a fermer

- quels champs doivent vivre en `ProtoConfig` versus `Credential`;
- quelle nomenclature de profils (`tlsProfile`, `businessIntegrationProfile`)
  est la plus coherente avec l'existant Gateway;
- faut-il distinguer des `ProtoConfig` `ebics-client` et `ebics-server` plus
  explicitement;
- faut-il supporter une creation automatique de `Rule` technique par defaut
  lors du provisioning EBICS.
