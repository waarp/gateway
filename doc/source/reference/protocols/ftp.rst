.. _ref-proto-ftp:

===
FTP
===

Malgré son nom de "File Transfer Protocol", le protocole FTP ne permet pas uniquement
la transmission de fichiers. En effet, FTP tient plus du protocole de gestion de
fichiers à distance, que du simple protocole de transfert. Néanmoins, dans le cadre de
la gateway seules les fonctionnalités pertinentes pour le MFT ont été implémentées.

Mode actif/passif
-----------------

La gateway supporte FTP en mode passif et actif, côté client et serveur.
Pour rappel, lorsqu'un client se connecte à un serveur FTP, le client ouvre une
connexion dite de "contrôle", permettant à celui-ci d'envoyer des commandes au
serveur. Lorsque le client souhaite effectuer un transfert de fichier, il doit
ouvrir une 2ème connexion dite de "données". Le mode actif/passif définie le
sens d'ouverture de cette 2ème connexion.

Par défaut, la gateway utilise le mode passif. En mode passif, c'est le client
qui ouvre la connexion de données. Le client envoie une commande ``PASV`` à
laquelle le serveur renvoie une adresse IP et un port auxquels le client peut
donc venir se connecter. À noter que la gateway supporte également le mode passif
étendu ``EPSV``, côté client et serveur, permettant le support de IPv6.

En mode actif, c'est le serveur qui initie la connexion de données. Le client
envoie une commande ``PORT`` accompagnée d'un adresse IP et d'un port sur lesquels
le serveur doit venir se connecter. Bien qu'historiquement, le mode actif est le
mode par défaut de FTP, ce mode a largement été abandonné en pratique car il
causait d'importants problèmes de sécurité.

Pour voir comment passer configurer les modes actif et passifs, consultez le
chapitre sur la :ref:`configuration FTP <proto-config-ftp>`. Par défaut,
le serveur FTP autorise les 2 modes de fonctionnement, et laisse le choix au
client. Le client FTP de la gateway, lui, privilégie le mode passif.

Commandes
---------

Le serveur FTP de la gateway supporte donc les commandes ``PASV``, ``EPSV`` et
``PORT`` pour le mode actif/passif.

L'initialisation de transfert se fait via les commandes ``STOR`` ou ``RETR``
(suivant le sens du transfert). Dans les 2 cas, étant donné que FTP n'offre pas
de mécanisme pour transmettre le nom de la règle à utiliser, c'est donc le chemin
du fichier qui est utilisé pour déterminer la règle. À noter que le serveur
supporte les transferts en mode ASCII et en mode Binaire, mais que le client
ne supporte que le mode Binaire.

Pour reprendre un transfert interrompu, la commande ``REST`` est supportée par
le client et le serveur.
Cette commande doit être envoyée avant la commande de transfert, et doit être
suivie du nombre de octets à partir duquel le transfert doit être reprit. À noter
que dans le cas d'un transfert "push" (``STOR``), ce nombre d'octets ne peut
être supérieur à la taille du fichier partiel (i.e. la quantité de données
déjà reçue), et ce, afin d'empêcher des trous dans le fichier. De même, dans
le cas d'un transfert "pull" (``RETR``), ce nombre d'octets ne peut être
supérieur à la quantité de données déjà envoyée.

.. warning:: Le serveur ne supporte pas la reprise pour les transferts "push" si
   ceux-ci n'ont pas été interrompus via la commande ``ABOR``. En effet, FTP ne
   contient aucun mécanisme permettant de distinguer une interruption de transfert
   d'une fin de transfert. Par conséquent, **tout transfert interrompu sans**
   ``ABOR`` **sera considéré comme terminé, ET NE POURRA PAS ÊTRE REPRIS**.
   Renseignez-vous sur le fonctionnement de votre client FTP pour savoir si ce
   dernier supporte cette fonctionnalité ou non. Dans le doute, n'interrompez
   pas les transferts en cours.

L'implémentation de FTP dans la gateway supporte également les commandes
``LIST`` et ``STAT`` permettant de récupérer des infos sur les fichiers présents
sur le serveur. Cependant, l'implémentation dans la gateway diffère des
implémentations classiques, car au lieu de lister les dossiers présents à
la racine, elle listera les règles de transfert, et les présentera sous forme
d'une arborescence reproduisant les "path" des règles. La commande ``SIZE``
permettant de juste récupérer la taille d'un fichier est également supportée.

Lister le contenu d'un de ces dossiers affichera la liste des fichiers pouvant
être récupéré avec la règle correspondante. Par conséquent, les dossiers des
règles de réception seront donc toujours vides (à moins qu'il existe une règle
d'envoi ayant le même *path*). Pour résumer, cela signifie que les fichiers
déposés sur le serveur ne seront pas visibles une fois le transfert terminé.

La commande de création de dossier ``MKD`` est également supportée.

|

Les autres commandes FTP non listées ci-dessus ne sont pas implémentées.
Notamment, la gateway ne supporte volontairement aucune commande de suppression
(que ce soit de fichier ou de dossier).

TLS
---

La gateway supporte la sécurisation des transferts FTP via TLS. Ce protocole
FTP over TLS, souvent appelé FTPS, ne doit pas être confondu avec SFTP
(SSH File Transfer Protocol) qui est un protocole complètement différent basé
sur SSH.

FTPS possède 2 modes de fonctionnement : explicite et implicite.

En mode TLS implicite, le client se connecte directement au serveur via une
connexion TLS. Cela implique que la gateway utilise des ports différents (et
donc des serveurs différent) pour FTP et FTPS (similairement à HTTP et HTTPS).

En mode TLS explicite, le client se connecte au serveur en FTP clair, puis
émet une commande ``AUTH TLS`` pour demander au serveur de passer en mode TLS.
Ce mode est aussi parfois appelé FTPeS.

Waarp-Gateway supporte les 2 modes de fonctionnement, que ce soit côté client ou
serveur. Dans le cas du TLS explicit, il est possible pour les serveur FTPS de
spécifier si TLS est obligatoire ou facultatif.

Authentification
----------------

Le serveur et le client FTP supportent tout deux l'authentification FTP standard
via mot de passe.

Dans le cas de FTPS, une authentification du client via certificat est également
possible, que ce soit en mode TLS explicite ou implicite.

Limitations
-----------

Dans le cadre du MFT, le protocole FTP a malheureusement plusieurs limitations:

1) FTP n'intégrant pas de mécanisme de synchronisation entre le client et le
   serveur durant un transfert, il est possible que l'émetteur du fichier
   termine le transfert bien avant le récepteur. Si une erreur survient donc du
   côté du récepteur alors que l'émetteur a déjà terminé le transfert, le client
   et le serveur seront donc en désaccord sur l'état final du transfert (l'un
   reportera le transfert comme terminé, alors que l'autre le reportera en erreur).
2) FTP n'intègre pas de mécanisme permettant au client d'envoyer une erreur au
   serveur. La seule commande s'en rapprochant un peu est la commande ``ABOR``.
   Cependant, tous les serveurs ne supportent pas cette commande. De plus, cette
   commande comporte une limitation notable qui est qu'elle n'a aucun effet si
   la commande de transfer a déjà terminé. Cela est particulièrement notable
   dans le cas des transferts en *pull*. Si le serveur a déjà terminé d'envoyer
   le fichier lorsque l'erreur survient, la commande ``ABOR`` du client n'aura
   aucun effet.
3) Si une erreur survient lors des post-traitements du serveur, celle-ci ne sera
   pas remontée au client. Il s'agit là plus d'une limitation de la librairie FTP
   utilisée que du protocole en lui-même.