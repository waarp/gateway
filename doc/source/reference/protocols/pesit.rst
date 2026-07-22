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

Waarp Gateway offre des modes de compatibilité pour le protocole PeSIT permettant
de communiquer avec les agents PeSIT ayant dévié des spécifications standards du
protocole.

Pour l'heure, le seul mode de compatibilité supporté est le mode ``axway``,
permettant de communiquer avec des agents PeSIT tels que *CFT* ou *SecureTransport*.
Ce mode de compatibilité doit être activé dans la :ref:`configuration protocolaire
<proto-config-pesit>` du partenaire ou du serveur local (selon le cas).
En mode de compatibilité ``axway``, le principal changement est que le champ de
*Filename* des requêtes de transfert est utilisé pour transmettre le nom de
la règle au lieu du nom de fichier. Celui-ci est, à la place, transmis via le
champ *FileLabel*.

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

.. _ref-pesit-articles:

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

Gateway stocke également le type de découpage (Fixe ou Variable, PI 31) sous forme
de transfert info sous le nom de clé ``__articlesFormat__``.

À l'inverse, pour les transferts en émission, il est possible d'utiliser les clés
``__articlesFormat__`` et ``__articlesLengths__`` pour spécifier le découpage des
articles. En l'absence de ces attributs, Gateway utilisera un découpage variable
avec des articles de 4096 octets max.

Pour conserver le découpage en articles d'un transfert à l'autre en cas de rebond,
pensez donc bien à activer l'option ``copyInfo`` de la tâche TRANSFER pour que la
clé soit copiée sur le nouveau transfert.

.. _ref-pesit-messages:

Messages
--------

Le protocol PeSIT intègre également la possibilité d'envoyer des messages hors
transfert. Ces messages sont typiquement utilisés pour confirmer (*acknowledge*)
qu'un transfert a bien terminé, même après plusieurs rebonds.

Il est possible d'informer Gateway qu'un transfert attend un *acknowledgement*.
Lorsqu'un transfert est marqué comme nécessitant un ACK, le transfert restera en
statut "RUNNING" (en cours) jusqu'à la réception de l'ACK. Une fois l'ACK reçu,
le transfert passera en statut "DONE" (terminé).

Un transfert en attente d'*acknowledgement* est marqué comme tel par la présence
de la clé ``__ackExpected__`` dans les *transfer info*. Une fois l'ACK reçu,
cette clé est supprimée et remplacée par ``__ackReceived__``  indiquant que l'ACK
a été reçu, et ``__ackReceivedOn__`` indiquant la date (format ISO 8601) à laquelle
l'ACK a été reçu.

Réciproquement, lorsque Gateway envoie un ACK pour un transfert, une clé ``__ackSent__``
sera ajoutée aux informations du transfert en question; ainsi que ``__ackSentOn__``,
``__ackSentTo__`` et ``__ackSentAs__`` indiquant respectivement la date, le partenaire
et le compte distant utilisés pour l'envoi de l'ACK.

Il existe 2 moyens pour marquer un transfert comme nécessitant un ACK :

1) En ajoutant directement la clé ``__ackExpected__`` avec la valeur ``true`` aux
   *transfer info* du transfert. Cela peut être fait dans la commande de création
   du transfert, dans la définition des tâches TRANSFER/PREREGISTER pour les rebonds,
   ou bien via la tâche SETINFO pour un transfert en cours.
2) Via l'option ``expectsAck`` de la :doc:`configuration protocolaire <../proto_config/pesit>`
   du partenaire PeSIT avec lequel le transfert prend place.

**Émission**

L'émission de messages de confirmation par Gateway se fait via la tâche
:doc:`SENDMESSAGE <../tasks/sendmessage>`. Celle-ci est typiquement placée en
fin de chaîne de post-traitements. À noter que celle-ci ne doit être utilisée
que si Gateway est bien le destinataire final du transfert. Si ce n'est pas le
cas, voir la section ci-dessous.

**Relay**

Lorsqu'un flux passe par Gateway mais que celle-ci n'est pas destinataire final
du flux, Gateway est capable de relayer le message de confirmation vers l'émetteur
original du flux.

À la réception de l'ACK, Gateway marquera les deux transferts (entrant et sortant)
comme terminés, et renverra le message reçu a la source du transfert entrant. Pour
pouvoir faire le lien entre les deux transferts, ceux-ci doivent impérativement
avoir le même *follow ID* (clé ``__followID__`` des transfer info). Il est donc
impératif de bien spécifier l'option ``copyInfo`` à ``true`` dans la tâche
TRANSFER programmant le rebond, sans quoi, Gateway ne pourra pas relayer le message.