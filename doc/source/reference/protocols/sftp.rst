.. _ref-proto-sftp:

####
SFTP
####

Bien que le protocole comporte de nombreuses fonctionnalités, seules celles
pertinentes pour le MFT ont été implémentées dans Waarp Gateway.

Commandes
=========

Il est donc possible d'initier un transfert via les commandes ``Put`` ou ``Get``
(suivant le sens du transfert). Dans les 2 cas, étant donné que SFTP n'offre pas
de mécanisme pour transmettre le nom de la règle à utiliser, c'est donc le chemin
du fichier qui est utilisé pour déterminer la règle.

L'implémentation de SFTP dans Waarp Gateway supporte également les commandes
``List`` et ``Stat`` permettant de récupérer les fichiers disponibles sur le
serveur. Cependant, l'implémentation dans Waarp Gateway diffère des
implémentations classiques, car elle masque les dossiers réels se trouvant sous
la racine du serveur. À la place, le serveur donne une liste de dossiers
correspondants aux ``path`` de toutes les règles utilisables par l'utilisateur.

Lister le contenu d'un de ces dossiers affichera la liste des fichiers pouvant
être récupéré avec la règle correspondante. Par conséquent, les dossiers des
règles de réception seront donc toujours vides (à moins qu'il existe une règle
d'envoi ayant le même ``path``). Pour résumer, cela signifie que les fichiers
déposés sur le serveur ne seront pas visibles une fois le transfert terminé.

Toutes les autres commandes SFTP, à savoir ``Setstat``, ``Rename``, ``Rmdir``,
``Mkdir``, ``Link``, ``Symlink``, ``Remove`` & ``Readlink`` ne sont pas
implémentées, car non-pertinentes pour le MFT.

Authentification
================

Authentification client
-----------------------

Le client SFTP est capable de s'authentifier via mot de passe et/ou via clé SSH.
La (ou les) forme d'authentification utilisées lorsque le client se connecte à
un serveur SFTP dépend de la configuration du compte (*RemoteAccount*).

De sont côté, le serveur SFTP de Waarp Gateway supporte également ces deux
formes d'authentification pour les clients s'y connectant. À noter que le serveur
vérifiera toutes les formes d'authentification fournies par le client.

Authentification serveur
------------------------

Le protocole SSH requiert que le serveur fournisse une clé d'hôte (*hostkey*)
lorsqu'un client s'y connecte. Par conséquent, le serveur SFTP de Waarp Gateway
doit impérativement être configuré avec au moins une clé SSH pour être utilisable
(il est possible de configurer un serveur avec plusieurs clés différentes).
Le serveur enverra toutes ses *hostkeys* aux clients qui s'y connectent.

Réciproquement, lorsqu'il se connecte à un partenaire SFTP, le client SFTP de
Waarp Gateway vérifiera qu'au moins une des clés SSH fournies par ce partenaire
soit connue.

Les algorithmes suivants sont acceptés pour la *hostkey* du serveur :

- ``ecdsa-sha2-nistp256``
- ``ecdsa-sha2-nistp384``
- ``ecdsa-sha2-nistp521``
- ``rsa-sha2-512``
- ``rsa-sha2-256``
- ``ssh-rsa``
- ``ssh-dss``
- ``ssh-ed25519``
