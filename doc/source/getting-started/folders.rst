.. _gestion_dossiers:

####################
Gestion des dossiers
####################

======
Résumé
======

Lors d'un transfert, l'utilisateur ne renseigne que la fin du chemin du fichier
à transférer. Le début de ce chemin est calculé par la *gateway* en fonction
des configurations :

- de la gateway, commune à tous les transferts
- du serveur, commune à tous les transferts vers/depuis ce :term:`serveur<server>`
  (non-valable pour les transferts clients)
- de la règle, commune à tous les transferts utilisant une même :term:`règle<rule>`

Les chemins plus spécifiques ont la priorité sur les chemins plus globaux.

.. warning:: Les chemins locaux, distants et temporaires d'une règle ne doivent
   pas être confondu avec le *path* de cette règle. Celui-ci sert uniquement à des
   fins d'identification de le règle, et n'intervient donc pas dans le calcul des
   chemin de fichiers.

**Exemple**

Prenons l'exemple d'un transfert entrant où la gateway agit comme serveur. Le
fichier à transférer est *file.txt*. Suivant les valeurs des différentes
configurations, le chemin de destination du fichier changera comme il suit :

+----------------+-----------------+----------------+----------------------+---------------+------------------------------+
| Dossier racine | Dossier 'in' de | Dossier racine | Dossier de réception | Dossier local | Chemin final                 |
| de la Gateway  | la Gateway      | du serveur     | du serveur           | de la règle   | du fichier                   |
+================+=================+================+======================+===============+==============================+
| /root          | in              |                |                      |               | /root/in/file.txt            |
+----------------+-----------------+----------------+----------------------+---------------+------------------------------+
| /root          | /in             |                |                      |               | /in/file.txt                 |
+----------------+-----------------+----------------+----------------------+---------------+------------------------------+
| /root          | /in             | serveur        |                      |               | /root/serveur/file.txt       |
+----------------+-----------------+----------------+----------------------+---------------+------------------------------+
| /root          | /in             | /serveur       |                      |               | /serveur/file.txt            |
+----------------+-----------------+----------------+----------------------+---------------+------------------------------+
| /root          | /in             | /serveur       | serv_recv            |               | /serveur/serv_in/file.txt    |
+----------------+-----------------+----------------+----------------------+---------------+------------------------------+
| /root          | /in             | /serveur       | /serv_recv           |               | /serv_in/file.txt            |
+----------------+-----------------+----------------+----------------------+---------------+------------------------------+
| /root          | /in             | /serveur       | /serv_recv           | rule_local    | /serveur/rule_local/file.txt |
+----------------+-----------------+----------------+----------------------+---------------+------------------------------+

Les transferts sortants fonctionnent de façon similaire, mais en remplaçant le
dossier *in* de la Gateway par son dossier *out*; et en remplaçant le dossier
de réception du serveur par son dossier d'envoi.

Les transferts client fonctionne également de façon similaire, mais sans les
dossiers spécifiques au serveur.

=======================
Explications détaillées
=======================

----------------
Dossiers gateway
----------------

Toute instance de *Waarp-Gateway* possède un dossier racine (appelé *GatewayHome*)
que l'utilisateur doit renseigner dans le fichier de configuration. Par défaut,
si l'utilisateur n'a pas renseigné de racine, le dossier courant est utilisé.
Par défaut, tous les chemins non-absolus seront relatif à cette racine.

C'est également dans le fichier de configuration que nos 3 dossiers mentionnés
plus haut sont définis, respectivement avec les options : le dossier 'in'
(*DefaultInDir*), le dossier 'out' (*DefaultOutDir*), et le dossier temporaire
(*DefaultTmpDir*). Lors d'un transfert, si aucun chemin plus spécifique n'est
renseigné, ces dossiers seront utilisés localement comme source ou destination
du fichier.

Par défaut, les valeurs 'in', 'out' et 'work' leur sont respectivement attribuées.
Ces chemins sont relatifs à la racine mentionnée plus haut. Il est également
possible de renseigner un chemin absolu si l'utilisateur ne souhaite pas que un
ou plusieurs des dossiers se trouvent sous la racine.

-------------------
Dossiers de serveur
-------------------

Lors de la création d'un :term:`serveur local<server>`, il est possible de
renseigner une racine au serveur (attribut *root*), ainsi que des sous-dossier
de réception (*receive*), d'envoi (*send*), et de réception temporaire (*tmpReceive*).

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

À la création d'une règle de transfert, il est possible d'attribuer des dossiers
spécifiques à cette règle. Ces dossier comprennent: un dossier local (*local*)
sur la *Gateway*, un dossier distant (*remote*) sur le(s) partenaire(s), et un
dossier local temporaire (*tmpLocal*).

Tous les fichiers transférés avec cette règle utiliseront donc ces dossiers. Pour
les règles d'envoi, les fichiers seront toujours récupérés depuis le dossier local,
puis déposés dans le dossier distant. À l'inverse, dans le cas des règles de reception,
les fichiers seront toujours récupérés depuis le dossier distant, puis déposés dans
le dossier temporaire local, avant d'être ensuite déplacés dans le dossier de réception
une fois le transfert terminé.

Ces dossier peuvent être des chemins absolus ou relatifs. Dans le second cas,
si la *gateway* agit comme client du transfert, ce chemin est alors relatif à
la racine. En revanche, si la *gateway* est serveur du transfert, alors tout chemin
relatif est considéré comme relatif à la racine du :term:`serveur local<server>`,
s'il en a une, ou bien à la racine de la *gateway* s'il n'en a pas.