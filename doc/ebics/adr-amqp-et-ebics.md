# ADR - AMQP natif, EBICS dans Gateway, et frontiere `Rule`

## Statut

Propose pour validation.

## Contexte

Le cadrage retenu pour l'integration d'EBICS dans Gateway est le suivant:

- Gateway porte les fonctions protocolaires et techniques;
- l'application metier porte les decisions non protocolaires;
- `AMQP 0.9.1` et `AMQP 1.0` sont introduits comme protocoles Gateway natifs,
  independants d'EBICS;
- EBICS vient ensuite s'appuyer sur ce socle d'integration.

Deux points doivent etre verrouilles avant tout developpement:

- `updateconf` doit suivre l'introduction des nouveaux protocoles;
- la notion de `Rule` ne doit pas casser le fonctionnement nominal d'EBICS.

## Decision 1 - `AMQP 0.9.1` et `AMQP 1.0` sont des protocoles Gateway natifs

Gateway introduira:

- un protocole `amqp091`;
- un protocole `amqp10`.

Ces protocoles:

- utilisent le cycle de vie nominal Gateway (`LocalAgent`, `Client`,
  `RemoteAgent`, `ProtoConfig`, supervision, admin);
- sont exploitables independamment d'EBICS;
- servent ensuite de socle de passe-plat asynchrone pour les usages EBICS.

Bibliotheques cible:

- `github.com/rabbitmq/amqp091-go` pour `AMQP 0.9.1`;
- `github.com/Azure/go-amqp` pour `AMQP 1.0`.

Contrainte de selection:

- pur Go uniquement;
- aucune dependance C ou runtime natif externe dans le chemin nominal.

## Decision 2 - `updateconf` entre explicitement dans le perimetre des evolutions

L'ajout d'un protocole n'est considere complet que si les objets concernes sont
transportables par:

- `export` / `import`;
- les formats JSON / YAML de backup;
- les archives traitees par `updateconf`.

Consequence:

- les nouveaux `Protocol` et `ProtoConfig` doivent etre representables dans le
  modele de sauvegarde;
- les exemples et documentations `updateconf` doivent etre enrichis;
- les campagnes de validation doivent inclure au moins un round-trip:
  `export -> archive -> updateconf -> import -> verification`.

## Decision 3 - `Rule` reste un objet Gateway, mais n'est pas un prerequis
## pour tous les ordres EBICS

La notion de `Rule` est un concept Waarp oriente transfert de fichier et
routage technique local/distant.

Elle reste pertinente pour:

- les transferts EBICS reellement orientes fichier;
- les politiques Waarp de chemin, repertoire temporaire et taches associees.

Elle n'est pas pertinente comme precondition universelle pour:

- les ordres administratifs EBICS;
- les ordres de consultation;
- les operations de cycle de vie protocolaire sans flux fichier.

La regle de conception devient donc:

- un objet `Transfer` Gateway n'est cree que pour les flux EBICS materialises
  comme transferts de fichier;
- ces flux conservent un `RuleID` comme aujourd'hui;
- les operations EBICS non orientees transfert ne sont pas forcees dans
  `Transfer` et disposent de leur propre persistance EBICS.

## Decision 4 - Pour EBICS, `Rule` devient un routage technique optionnel

Pour les flux fichier EBICS, `Rule` est interpretee comme:

- une politique technique de depot/reception;
- une politique d'autorisation Waarp;
- une politique optionnelle de taches avant/apres transfert.

Elle n'est pas interpretee comme:

- un objet metier EBICS;
- une expression du contrat banque;
- une condition de validite du protocole.

Si un flux EBICS ne necessite pas de routage particulier, une regle technique
simple et standardisable pourra etre utilisee.

## Consequences

Benefices:

- on preserve le fonctionnement nominal de Gateway la ou il apporte de la
  valeur;
- on n'ecrase pas EBICS sous une abstraction `Rule` artificielle;
- on garde une frontiere claire entre transfert de fichier et operation
  protocolaire;
- on evite qu'un protocole nouveau soit incomplet du point de vue
  exploitation/import-export.

Contraintes:

- il faut distinguer clairement `Transfer EBICS` et `Operation EBICS`;
- il faut prevoir des jeux de tests `updateconf` pour les nouveaux protocoles;
- l'API et la CLI doivent rendre visible qu'un ordre EBICS administratif n'est
  pas un transfert.

## Validation attendue dans les spikes

- un `Transfer` EBICS fichier peut etre cree sans tordre le modele `Rule`;
- un ordre EBICS administratif peut etre journalise sans passer par `Transfer`;
- une configuration `amqp091`, `amqp10` et `ebics` survit a un cycle
  `export/import/updateconf`;
- l'administration comprend sans ambiguite quand `Rule` est requise et quand
  elle ne l'est pas.
