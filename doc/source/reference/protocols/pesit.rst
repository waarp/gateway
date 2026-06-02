.. _ref-proto-pesit:

PeSIT
=====

Authentification
----------------

L'authentification du client se fait via un mot de passe fourni au moment de la
connexion. Optionnellement, le serveur peut également s'authentifier via un
mot de passe donné en réponse à celui du client.

Il est à noter cependant que le protocole PeSIT impose une longueur maximale de
8 caractères pour les mots de passe. De plus, seuls les caractères ASCII sont
acceptés. Si le mot de passe ne remplie pas ces critères, il sera refusé au
moment de son insertion en base de données.

Dans le cas où la connexion se fait par dessus un tunnel TLS, l'authentification
via certificat est également possible pour le client et le serveur.

L'authentification client se configure de la même manière que n'importe quel
autre protocole utilisant l'authentification par mot de passe.

Pour l'authentification serveur, le login doit être renseigné dans la :ref:`
configuration protocolaire <proto-config-pesit>`. Le mot de passe, lui, doit
être renseigné dans un :ref:`identifiant <reference-auth-methods>` de type
*"password"* rattaché au partenaire/serveur.

Pré-connexion
-------------

Le protocole PeSIT inclue une étape de pré-connexion survenant avant l'établissement
même de la connexion PeSIT. Cette étape est optionnelle et peut être désactivée
via la :ref:`configuration protocolaire <proto-config-pesit>` du partenaire ou
du serveur.

Cette pré-connexion inclue notamment une étape d'authentification supplémentaire
pour le client. Lorsque la Gateway est client, ces identifiants de pré-connexion
peuvent être renseignés via des :ref:`identifiants <reference-auth-methods>` de
type *"pesit_pre-connection_auth"* rattachés au compte distant utilisé pour le
transfert. Lorsque la Gateway est serveur, tous les identifiants de pré-connexion
seront acceptés quels qu'ils soient, ceux-ci étant redondants avec la procédure
d'authentification standard décrite ci-dessus.


Profils
-------

Waarp Gateway ne supporte que le profil "non SIT" de PeSIT, et uniquement par
dessus une connexion TCP (ou TLS pour PeSIT-TLS). Les autres profils de PeSIT
ne sont pas supportés.

Checkpoints et *restart*
------------------------

Le protocol PeSIT intègre un mécanisme de *checkpoints* lors d'un transfert.
Dans les faits, l'émetteur du fichier peut périodiquement demander au récepteur
de confirmer qu'il a bien reçu l'intégralité des données jusqu'à la position
mentionnée dans la demande de checkpoint. La fréquence de ces demandes de
checkpoint est négociée entre le client et le serveur, et elle peut être
configurée via leurs :ref:`configurations protocolaires <proto-config-pesit>`
respectives. Cette configuration protocolaire spécifie également le nombre de
checkpoints pouvant rester sans réponse. Au delà de cette limite, l'émetteur
cessera l'envoi de nouvelles données jusqu'à ce que le récepteur réponde à ses
demandes de checkpoints.

À tout moment, en cas d'erreur, chacun des deux partenaires du transfert peut
demander à l'autre de faire un *restart* du transfert, et de donc reprendre ce
transfert à un checkpoint préalablement validé. Il est à noter que bien que
Waarp Gateway permette à un partenaire externe de faire un *restart* de transfert
quand il le souhaite (y compris pendant le transfert lui-même), la Gateway ne
demandera jamais automatiquement de faire un *restart* elle-même, la politique
de Waarp étant que la reprise de transfert est une prérogative de l'utilisateur.

Ces 2 fonctionnalités (les checkpoints et les restarts) peuvent être désactivés
si besoin via les :ref:`configurations protocolaires <proto-config-pesit>` des
clients, serveurs et partenaires.

Modes de compatibilité
----------------------

Waarp Gateway supporte deux modes de fonctionnement pour le protocole PeSIT,
configurables via la :ref:`configuration protocolaire <proto-config-pesit>` du
partenaire ou du serveur local :

- **Mode standard** (``standard``, par défaut) : le chemin du fichier distant
  contient le nom de la règle en préfixe (ex: ``regle/fichier.txt``). Le serveur
  identifie la règle à appliquer par correspondance de préfixe sur le chemin.

- **Mode Axway** (``axway``) : mode de compatibilité avec les agents PeSIT tels
  que *CFT* ou *SecureTransport*. Le champ *Filename* (PI 12) est utilisé pour
  transmettre le nom de la règle au lieu du chemin de fichier. Celui-ci est, à la
  place, transmis via le champ *FileLabel* (PI 37).

.. _ref-proto-pesit-patterns:

Pull avec pattern (glob)
------------------------

.. versionadded:: 0.16.0

En mode pull (réception), le client peut envoyer un nom de fichier contenant des
**caractères génériques** (``*`` et ``?``) dans sa requête de sélection. Le
serveur résout alors le pattern pour trouver un fichier correspondant :

1. **Transferts pré-enregistrés** : le serveur cherche d'abord parmi les transferts
   ayant le statut ``AVAILABLE`` dont le nom de fichier correspond au pattern
   (via ``path.Match``). Le premier transfert correspondant (le plus ancien par
   date de soumission) est sélectionné.

2. **Système de fichiers** : si aucun transfert pré-enregistré ne correspond, le
   serveur effectue un glob sur le répertoire d'envoi de la règle et sélectionne
   le premier fichier correspondant.

Le nom de fichier résolu est renvoyé au client dans la réponse de sélection.
Le client crée alors le fichier local avec le nom réel (et non le pattern).

**Exemple** : un client envoie une requête de pull avec le pattern ``data-*.txt``.
Si un transfert AVAILABLE existe avec le nom ``data-001.txt``, il sera sélectionné.
Sinon, si un fichier ``data-001.txt`` existe dans le répertoire d'envoi de la règle,
celui-ci sera servi.

**Cas de correspondances multiples** :

- Les transferts pré-enregistrés sont consommés un par un : chaque SELECT avec le
  même pattern retourne le prochain transfert AVAILABLE correspondant.
- Pour les fichiers du système de fichiers, il est recommandé d'utiliser une tâche
  ``DELETE`` en post-traitement sur la règle. Ainsi, après chaque envoi, le fichier
  est supprimé et le prochain SELECT avec le même pattern sélectionne le fichier
  suivant.

.. note::

   Cette fonctionnalité n'est disponible qu'en mode ``standard``. En mode ``axway``,
   le nom de fichier ne transite pas dans le même champ PeSIT et le pattern matching
   n'est pas applicable.

F.MESSAGE
---------

.. versionadded:: 0.16.0

Le protocole PeSIT supporte l'envoi de messages (F.MESSAGE) entre partenaires
connectés, en dehors de tout transfert de fichier. Ce mécanisme est typiquement
utilisé pour les accusés de réception *Store and Forward* et les messages
applicatifs libres.

Un F.MESSAGE peut contenir :

- Un texte libre (contenu du message)
- Un identifiant de transfert associé (PI 13)
- Un identifiant client (PI 61) et un identifiant banque (PI 62)

L'envoi de F.MESSAGE se fait depuis l'état "connecté" (aucun fichier sélectionné).
Le serveur peut accepter ou refuser le message en retournant un code diagnostic
approprié.

Transferts multi-fichiers
-------------------------

Le protocole PeSIT permet d'effectuer plusieurs transferts de fichiers sur une
même connexion TCP/TLS (multi-sélection). Dans Waarp Gateway, chaque transfert
correspond à une sélection de fichier (F.CREATE ou F.SELECT) distincte au sein
de la même session PeSIT.

Le client peut donc enchaîner plusieurs opérations push et/ou pull sur une même
connexion, réduisant ainsi le coût d'établissement de connexion et d'authentification.

Texte libre
-----------

Le protocole PeSIT permet au client et au serveur d'échanger des champs de texte
libre, une première fois à la connexion, puis une seconde fois lors de la sélection
du fichier de transfert. Waarp Gateway permet aux utilisateurs le souhaitant de
peupler ces champs qui peuvent être parfois utilisés par des applications tierces.

Les valeurs de ces champs sont stockées dans les :term:`infos de transfert` via
les clés spéciales suivantes :

- ``__clientConnFreetext__`` pour le texte envoyé par le client à la connexion
- ``__clientTransFreetext__`` pour le texte envoyé par le client à la sélection du fichier
- ``__serverConnFreetext__`` pour le texte envoyé par le serveur à la connexion
- ``__serverTransFreetext__`` pour le texte envoyé par le serveur à la sélection du fichier

Ces valeurs étant stockées dans les :term:`infos de transfert`, il est donc possible
de référencer ces valeurs dans des traitements via les :ref:`marqueurs de substitution
<reference-tasks-substitutions>`.

Attributs PeSIT
---------------

En plus du texte libre, le protocole PeSIT permet la transmission de divers attributs
et informations. Comme le texte libre, ces informations sont stockées par la Gateway
sous forme :term:`d'infos de transfert<infos de transfert>` avec des clés spéciales
réservées. Les clés correspondantes pour ces attributs sont :

- ``__fileEncoding__`` pour l'encodage du fichier (PI 16)
- ``__fileType__`` pour le type de fichier (PI 11)
- ``__organization__`` pour l'organisation du fichier (PI 33)
- ``__customerID__`` pour l'identifiant de client (PI 61)
- ``__bankID__`` pour l'identifiant de banque (PI 62)

À noter que les 3 premiers attributs voyagent toujours dans le même sens que le fichier
transféré (donc de l'émetteur vers le récepteur), alors que les 2 derniers voyagent
toujours dans le sens de la connexion (donc du client vers le serveur).

Comme pour le texte libre, ces informations peuvent être référencées dans les traitements
en utilisant leurs clés respectives.

Articles
--------

Le protocole PeSIT permet techniquement d'envoyer plusieurs "articles" au sein
d'un même transfert. Bien que cela puisse potentiellement permettre de transférer
plusieurs fichiers en un seul transfert, il est à noter **qu'un article n'est pas
équivalent à un fichier**. Notamment, les fichiers de grande taille seront
quasi-systématiquement découpés en plusieurs articles.

Voici donc comment Gateway gère ce découpage en articles :

En réception, tous les articles envoyés par l'émetteur seront stockés dans un
même fichier sur le disque. Le découpage en article sera lui stocké dans les
:term:`infos de transfert` sous le nom de clé ``__articlesLengths__``.
Cet attribut prendra la forme d'une liste JSON d'entiers spécifiant la taille
(en octets) de chaque article du transfert.

À l'inverse, pour les transferts en émission, si la clé ``__articlesLengths__``
est présente dans les infos de transfert, alors sa valeur sera utilisée pour le
découpage en articles. En l'absence de cet attribut, un découpage automatique
minimisant le nombre d'articles sera utilisé par défaut.

Pour conserver le découpage en articles d'un transfert à l'autre en cas de rebond,
pensez donc bien à activer l'option ``copyInfo`` de la tâche TRANSFER pour que la
clé soit copiée sur le nouveau transfert.