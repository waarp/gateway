# Spike C - Validation `export / import / updateconf`

## Statut

Conclu.

Decision proposee: `GO`.

## 1. Objet

Ce spike verifie si l'ajout des protocoles `amqp091`, `amqp10` et `ebics`
reste compatible avec la chaine de configuration existante de Gateway:

- `export`
- `import`
- formats JSON / YAML
- `updateconf`

## 2. Constat sur l'existant

La chaine actuelle est deja largement generique.

### 2.1 `export / import`

Le modele de sauvegarde transporte deja:

- `Client.Protocol` + `Client.ProtoConfig`
- `RemoteAgent.Protocol` + `RemoteAgent.Configuration`
- `LocalAgent.Protocol` + `LocalAgent.Configuration`

References utiles:

- [file.go](/c:/MonProjet/Waarp-Gateway/pkg/backup/file/file.go)
- [client-export.go](/c:/MonProjet/Waarp-Gateway/pkg/backup/client-export.go)
- [import.go](/c:/MonProjet/Waarp-Gateway/pkg/backup/import.go)

Conclusion:

- les nouveaux protocoles peuvent entrer naturellement dans le format existant;
- aucun changement de schema de sauvegarde n'apparait necessaire pour porter
  `amqp091`, `amqp10` et `ebics`.

### 2.2 `updateconf`

Deux mecanismes existent:

- un utilitaire distribue dans [updateconf.go](/c:/MonProjet/Waarp-Gateway/dist/updateconf/updateconf.go)
- une tache Gateway dans [updateconf.go](/c:/MonProjet/Waarp-Gateway/pkg/tasks/updateconf.go)

Les deux reposent sur un principe simple:

- extraire un fichier `<GatewayName>.json` depuis une archive ZIP;
- appeler ensuite l'import existant.

Conclusion:

- `updateconf` ne connait pas les protocoles individuellement;
- il herite donc naturellement de la generisation de `backup.Import`.

## 3. Point favorable majeur

La representation des protocoles dans la sauvegarde est deja suffisamment
abstraite:

- un nom de protocole;
- une map `ProtoConfig`/`Configuration`.

Cela signifie que l'ajout de `amqp091`, `amqp10` et `ebics` releve d'abord:

- de la validation de `ProtoConfig`;
- de la documentation;
- des exemples de configuration;
- des tests de round-trip.

Pas d'une refonte du mecanisme `updateconf`.

## 4. Points d'attention

### 4.1 Nommage des fichiers dans `updateconf`

Le chemin nominal de `updateconf` attend:

- un fichier `<GatewayName>.json`;
- et optionnellement `fw.json` / `get-file.list`.

Consequence:

- les exemples de provisioning des nouveaux protocoles devront etre publies au
  format compatible avec cette convention;
- il ne faut pas supposer que `updateconf` charge plusieurs fichiers
  protocolaires separes.

### 4.2 Validation des `ProtoConfig`

La vraie robustesse du dispositif repose sur:

- `ConfigChecker.CheckClientConfig`
- `ConfigChecker.CheckPartnerConfig`

Donc le risque n'est pas dans `updateconf` lui-meme, mais dans:

- la qualite des structures de config des nouveaux protocoles;
- la stabilite de leur validation.

### 4.3 JSON / YAML

L'import lit deja:

- `json` par defaut;
- `yaml` si l'extension du fichier le demande.

Conclusion:

- les nouveaux protocoles doivent etre documentes dans les deux formes;
- il faut verifier que les `ProtoConfig` restent lisibles et non ambigus en
  YAML.

## 5. Decision du spike

`GO`

Motif:

- aucune inadequation structurelle n'a ete identifiee;
- la chaine existante est suffisamment generique pour absorber les nouveaux
  protocoles.

## 6. Travaux a prevoir

Ce spike ouvre un backlog court mais obligatoire:

- ajouter des exemples `amqp091`, `amqp10` et `ebics` en JSON/YAML;
- ajouter des tests de round-trip `export -> import`;
- ajouter un test `ZIP -> updateconf`;
- verifier que les nouveaux `ProtoConfig` apparaissent proprement dans les
  sauvegardes;
- mettre a jour la documentation `updateconf`.

## 7. Resultat architectural

L'ajout de nouveaux protocoles n'est pas bloque par l'administration de
configuration.

Autrement dit:

- `updateconf` n'est pas un risque d'architecture majeur;
- c'est un point d'industrialisation et de validation produit.
