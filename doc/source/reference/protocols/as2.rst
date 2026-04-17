===
AS2
===

AS2 est un protocole transitant par dessus HTTP en ajoutant des fonctionnalités
de sécurité et de contrôle de données. Gateway implémente la version 1.1 du
protocole telle que décrite dans la :rfc:`4130`.

Dans Waarp Gateway, AS2 se décline en 2 protocoles distincts :

- *AS2 over HTTP*, appelé "as2", opérant par dessus une connexion HTTP claire
- *AS2 over HTTPS*, appelé "as2-tls" opérant par dessus une connexion HTTPS chiffrée

Présentation générale
---------------------

Le premier point important à noter à propos de AS2 est qu'il ne permet que de
faire des transferts dans la direction client -> serveur (aussi appelé *push*
ou *upload*). **Il est protocolairement impossible de transférer un fichier
depuis un serveur vers un client** (aussi appelé *pull* ou *download*).

Le client envoie une requête HTTP *POST* au serveur contenant le payload (le fichier)
dans le corps de la requête. Ce payload est optionnellement enrobé dans un
conteneur PKCS7 (voir section "Cryptographie" ci-dessous), lui-même enrobé dans
un conteneur S/MIME.

Une fois le payload traité par le serveur, celui-ci renvoie un acquittement
appelé MDN pour *Message Disposition Notification*. Cet acquittement peut être
synchrone ou asynchrone (voir section "Acquittement" ci-dessous).

Limitations
-----------

Dû à des spécificités d'implémentation du format PKCS7, Gateway est contrainte de
stocker intégralement les fichiers envoyés et reçu en mémoire vive. Pour cette
raison, Gateway impose une limite à la taille des fichiers envoyés et reçus via
AS2.

Par défaut, cette limite est de 1Mo. Pour la changer, utilisez l'option
``maxFileSize`` des :doc:`configurations protocolaires <../proto_config/as2>`
client ou serveur (suivant le cas).

Authentification
----------------

Gateway implémente l'authentification AS2 telle que décrite dans la spécification
du protocole.

Celle-ci se base sur les entêtes ``AS2-To`` et ``AS2-From`` pour l'identification,
combinés avec des paires de clés privées/publiques servant à signer les messages
envoyés par le client et le serveur.

Gateway implémente également une authentification HTTP basique permettant
d'authentifier la couche HTTP (voir la :rfc:`7617` pour plus de détails).

Cryptographie
-------------

Le principal attrait du protocol AS2 est sont intégration native de fonctionnalités
cryptographiques pour sécuriser les données envoyées. Ces fonctionnalités sont
regroupées en 3 parties :

- le chiffrement du payload
- la signature du payload
- la signature du MDN d'acquittement

Il est à noter que ces fonctionnalités requierent des certificats x509 pour
opérer, et ce, même si le transfert ne se fait pas via TLS/HTTPS.

Cette configuration peut être difficile a appréhender, mais elle peut être
résumée de la façon suivante :

- Pour la signature cryptographique, un agent signe *en son propre nom* (pour
  certifier qu'il a bien émit les données jointes). Il a donc besoin pour cela
  d'un certificat qui lui est propre, et de la clé privée qui va avec.
- Pour le chiffrement cryptographique, un agent A chiffre les données *à l'intention
  d'un autre agent B* (pour que personne d'autre ne puisse les lires). Pour ce faire,
  l'agent A a donc besoin du certificat de l'agent B, afin de chiffrer les données
  avec la clé publique de B.

Client
^^^^^^

Dans le cas où Gateway est client du transfert, la configuration se fait au
niveau du partenaire concerné par le transfert. Les options ``encryptionAlgorithm``
et ``signatureAlgorithm`` de la :doc:`configuration protocolaire partenaire
<../proto_config/as2>` permettent d'activer respectivement le chiffrement et la
signature du payload, et de choisir également l'algorithme utilisé pour ce faire.

Il est important de noter que, pour pouvoir signer le payload, Gateway a besoin
d'un certificat x509 et de la clé privée allant avec. Ceux-ci doivent être
attachés au **compte distant** utilisé pour le transfert sous forme d'un
identifiant de type ``tls_certificate`` (voir le :doc:`document <../auth_methods>`
sur les méthodes d'authentification pour plus de détails).

Similairement, pour pouvoir chiffrer le payload, Gateway a besoin d'un certificat
x509 (sans clé privée cette fois ci) rattaché au **partenaire distant** avec
lequel le transfert va avoir lieu, sous forme d'un identifiant de type
``trusted_tls_certificate``.

Serveur
^^^^^^^

Dans le cas où Gateway est serveur du transfert, la configuration protocolaire
du serveur permet, via l'option ``mdnSignatureAlgorithm`` d'activer et de choisir
l'algorithme pour la signature du MDN d'acquittement.

Pour pouvoir signer le MDN, Gateway a besoin d'un certificat x509 rattaché au
**serveur local** recevant la requête, sous forme d'un identifiant de type
``tls_certificate``.

Acquittement
------------

AS2 intègre nativement un mécanisme d'acquittement des transferts appelé MDN
(pour *Message Disposition Notification*). Cet acquittement peut être synchrone
ou asynchrone. Par défaut, Gateway demande des acquittements synchrones.

À noter que le choix du mode d'acquittement revient entièrement au client. Il
n'y a donc pas de configuration possible lorsque Gateway agit comme serveur,
celle-ci se contentera de suivre ce que le client lui a demandé. Les sections
suivantes ne concernent donc que les cas où Gateway agit comme client du transfert.

Synchrone
^^^^^^^^^

Un MDN synchrone sera envoyé immédiatement par le serveur en réponse de la
requête HTTP originale du client. Il se fera donc dans la même connexion que la
requête, et le client devra obligatoire attendre l'acquittement pour poursuivre.
Il s'agit du comportement par défaut de Gateway, et ne nécessite aucune
configuration particulière.

Asynchrone
^^^^^^^^^^

Pour spécifier à Gateway d'utiliser des MDNs asynchrones, la configuration se
fait de nouveau au niveau de la :doc:`configuration protocolaire du partenaire
<../proto_config/as2>`, en l'occurrence via l'option ``asyncMDNAddress``. Si
cette option est présente, elle indiquera au partenaire d'envoyer son
acquittement à l'adresse spécifiée.

Par défaut, il est présumé que cet acquittement sera géné par une application
tierce, différente de Gateway. Dans ce cas de figure, Gateway poursuivra le
transfert sans attendre d'acquittement de la part du partenaire.

Il est possible de dire à Gateway de gérer elle-même l'acquittement asynchrone
via l'option ``handleAsyncMDN`` de la configuration protocolaire. Si cette option
est activée, Gateway écoutera à l'adresse fournie dans l'option ``asyncMDNAddress``,
et attendra un acquittement du partenaire avant de terminer les transferts.
Veillez à ce que l'adresse et le port en question soit libres, et pensez à mettre
des exceptions dans votre pare-feu si nécessaire.