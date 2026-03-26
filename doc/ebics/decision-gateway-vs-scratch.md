# Decision Gateway vs From Scratch

## 1. Question

La question n'est pas seulement de savoir si la librairie EBICS peut etre
branchee dans Gateway.

La vraie question est:

- est-ce que Gateway reste le bon conteneur architectural pour EBICS;
- ou bien est-ce qu'EBICS impose un runtime suffisamment specifique pour
  justifier une application dediee.

## 2. Ce que Gateway apporte deja

Gateway apporte deja:

- un registre de protocoles modulaires;
- un cycle de vie standardise des services client/serveur;
- une administration mature REST/CLI/UI;
- une persistance deja structuree pour agents, comptes, credentials, regles et
  transferts;
- un pipeline de traitements et d'historisation utile pour les flux fichier;
- un socle d'exploitation deja present.

Conclusion:

- si EBICS peut rester "un protocole de plus" dans une plateforme MFT,
  Gateway est potentiellement un bon candidat.

## 3. Ce qui peut faire echouer l'option Gateway

Les points de friction reels sont:

- Gateway est fortement centre sur les transferts de fichiers;
- EBICS ne se reduit pas a des transferts de fichiers;
- les workflows d'initialisation, de rotation de cles et de RTN n'existent pas
  aujourd'hui dans Gateway;
- le risque est de vouloir tout forcer dans `Transfer`, `Pipeline` et
  `TransferInfo`.

Conclusion:

- si l'integration impose de tordre le coeur Gateway, alors il faut envisager
  serieusement une application dediee.

## 4. Hypothese de travail recommandee

L'hypothese la plus defendable a ce stade est:

- Gateway est un bon candidat si l'objectif est `MFT + EBICS`;
- Gateway devient un mauvais candidat si l'objectif devient une plateforme
  EBICS tres specialisee ou a dominante metier bancaire.

Cette hypothese devient plus favorable a Gateway si le perimetre est recentre
sur:

- les automatismes protocolaires;
- la rotation des cles;
- la recuperation automatisee des reports;
- RTN et l'auto-pull;
- la gestion des signatures au sens protocolaire;
- un passe-plat vers l'application metier pour les decisions non protocolaires.

## 5. Regle de go / no-go

On continue sur Gateway si les lots 1 et 2 montrent que:

- `ebics` s'integre proprement au registre de protocoles;
- les stores durables s'ajoutent sans refondre le coeur;
- les ordres non fichier peuvent vivre hors `Transfer`;
- les nouvelles tables et services restent modulaires;
- le socle d'administration et d'exploitation reste un accelerateur.
- la logique metier peut rester hors Gateway sans contorsion.

On reconsidere `from scratch` si les lots 1 et 2 montrent que:

- `Transfer`, `Pipeline` ou `Controller` doivent etre modifies en profondeur;
- les APIs existantes deviennent un frein plus qu'un accelerateur;
- la persistance EBICS devient plus importante que le modele Gateway existant;
- l'essentiel de la valeur se deplace vers des workflows bancaires specifiques.

## 6. Decision provisoire

La bonne posture est donc:

- ne pas coder trop tot;
- utiliser les lots 1 et 2 comme preuve architecturale;
- trancher seulement apres cette preuve.

En l'etat actuel de l'analyse, Gateway reste un candidat plausible, mais pas
encore demontre.

Avec le recentrage "Gateway = moteur protocolaire / application metier = moteur
de decision", Gateway devient toutefois un candidat nettement plus credible.
