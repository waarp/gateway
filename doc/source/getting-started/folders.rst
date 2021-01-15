.. _gestion_dossiers

####################
Gestion des dossiers
####################

======
Résumé
======

Lors d'un transfert, l'utilisateur ne renseigne que le nom du fichier à transférer.
Le dossier où se trouve se fichier est ensuite calculé par la *gateway* en fonction
de plusieurs paramètres. On distingue 3 types de dossiers différents :

- le dossier source (*out*) : c'est là que la *gateway* va chercher le fichier,
  que ce soit en local ou à distance
- le dossier de travail (*work*) : c'est là que la *gateway* stocke les fichiers
  en cours de réception (par conséquent ce dossier n'est utilisé que pour les
  transferts entrants)
- le dossier destination (*in*) : c'est là que la *gateway* va déposer le fichier,
  que ce soit en local ou à distance

Ces chemins peuvent être définis à 3 niveaux différent, du plus large au plus
spécifique :

- gateway, commun à tous les transferts
- serveur, commun à tous les transferts vers/depuis un :term:`serveur<server>`
- règle, commun à tous les transferts utilisant une même :term:`règle<rule>`

Les chemins plus spécifiques ont la priorité sur les chemins plus globaux.

.. warning:: Les chemins `inPath`, `outPath` & `workPath` d'une règle ne doivent
   pas être confondu avec le `path` de la règle. Celui-ci sert uniquement à des
   fins d'identification de le règle, et n'intervient donc pas dans le calcul des
   chemin de fichiers.

**Exemple**

Prenons l'exemple d'un transfert entrant où la gateway agit comme serveur. Le
fichier à transférer est `file.txt`. Suivant les différentes configurations
possibles, le chemin de destination du fichier changera comme il suit :

+----------------+------------+----------------+------------+----------+---------------------------+
| Racine gateway | Gateway in | Racine serveur | Serveur in | Règle in | Chemin final              |
+================+============+================+============+==========+===========================+
| /root          |            |                |            |          | /root/file.txt            |
+----------------+------------+----------------+------------+----------+---------------------------+
| /root          | in         |                |            |          | /root/in/file.txt         |
+----------------+------------+----------------+------------+----------+---------------------------+
| /root          | /in        |                |            |          | /in/file.txt              |
+----------------+------------+----------------+------------+----------+---------------------------+
| /root          | /in        | serveur        |            |          | /root/serveur/file.txt    |
+----------------+------------+----------------+------------+----------+---------------------------+
| /root          | /in        | /serveur       |            |          | /serveur/file.txt         |
+----------------+------------+----------------+------------+----------+---------------------------+
| /root          | /in        | /serveur       | serv_in    |          | /serveur/serv_in/file.txt |
+----------------+------------+----------------+------------+----------+---------------------------+
| /root          | /in        | /serveur       | /serv_in   |          | /serv_in/file.txt         |
+----------------+------------+----------------+------------+----------+---------------------------+
| /root          | /in        | /serveur       | /serv_in   | rule_in  | /serveur/rule_in/file.txt |
+----------------+------------+----------------+------------+----------+---------------------------+

Les transferts sortants fonctionnent de façon similaire, mais en utilisant les
dossiers `out` au lieu de `in`.

=======================
Explications détaillées
=======================

----------------
Dossiers gateway
----------------

Toute instance de *Waarp-Gateway* possède un dossier racine (appelé *GatewayHome*)
que l'utilisateur doit renseigner dans le fichier de configuration. Par défaut,
si l'utilisateur n'a pas renseigné de racine, le dossier courant est utilisé.
Tous les chemins non-absolus seront relatif à cette racine.

C'est également dans le fichier de configuration que nos 3 dossiers mentionnés
plus haut sont définis, respectivement avec les options : *InDirectory*,
*OutDirectory* et *WorkDirectory*. Lors d'un transfert, si aucun chemin plus
spécifique n'est renseigné, ces dossiers seront utilisés comme source ou
destination du fichier.

Par défaut, les valeurs 'in', 'out' et 'work' leur sont respectivement attribuées.
Ces chemins sont relatifs à la racine mentionnée plus haut. Il est également
possible de renseigner un chemin absolu si l'utilisateur ne souhaite pas que un
ou plusieurs des dossiers se trouvent sous la racine.

-------------------
Dossiers de serveur
-------------------

Lors de la création d'un :term:`serveur local<server>`, il est possible de
renseigner une racine au serveur (attribut `root`), ainsi que des dossier `in`,
`out` et `work`.

La racine peut être un chemin absolu ou relatif. Dans le second cas, le chemin
sera relatif à la racine de la gateway. De même, les sous-dossiers peuvent être
absolus ou relatif, auquel cas, il seront relatif à la racine du serveur.

Étant rattachés à un serveur local, ces dossiers ne sont utilisés que pour les
transferts où la *gateway* agit comme serveur. Si les dossiers d'un serveur sont
définis, ils supplantent les dossiers globaux de la *gateway* lors du calcul de
chemin du fichier.

-----------------
Dossiers de règle
-----------------

À la création d'une règle de transfert, il est possible d'ajouter des chemins
*in*, *out* et *work* à celle-ci. Tous les fichiers transférés avec cette règle
utiliseront donc ces dossiers. Contrairement aux chemins de niveaux supérieurs,
les chemins d'une règle sont toujours relatifs.

Si la *gateway* agit comme client du transfert, ce chemin est alors relatif à
la racine. Si la *gateway* est serveur du transfert, alors le chemin est relatif
à la racine du :term:`serveur local<server>` s'il en a une, ou bien à la racine
de la *gateway* est utilisée à la place s'il n'en a pas.