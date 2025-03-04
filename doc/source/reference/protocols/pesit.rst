.. _ref-proto-pesit:

PeSIT *(beta)*
==============

.. warning:: L'implémentation de PeSIT est actuellement en **beta**. En effet,
   celle-ci n'a, pour l'heure, pas pu être testée avec d'autres agents PeSIT.
   Par conséquent, nous ne pouvons donner de garantie absolue que cette implémentation
   soit compatible avec tous les agents PeSIT tiers du marché.

   Nous vous prions de bien vouloir nous remonter tout problème de compatibilité
   que vous pourriez observer lors de votre utilisation afin que nous puissions
   les corriger.

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
peuvent être renseignés via des ref:`identifiants <reference-auth-methods>` de
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

Il est à noter que Waarp Gateway ne permettant pas de pré-renseigner des informations
de transfert pour les transfert serveur, les clés ``__serverConnFreetext__`` et
``__serverTransFreetext__`` ne seront utilisées que par le client PeSIT pour stocker
le texte envoyé par un serveur PeSIT tier. Le serveur PeSIT de Waarp Gateway ne
renverra jamais de texte.

Ces valeurs étant stockées dans les :term:`infos de transfert`, il est donc possible
de référencer ces valeurs dans des traitements via les :ref:`marqueurs de substitution
<reference-tasks-substitutions>`.