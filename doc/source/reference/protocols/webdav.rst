======
Webdav
======

WebDAV est une extension du protocole HTTP permettant la gestion de fichiers à
distance. Dans le cadre de Waarp Gateway, seuls les aspects pertinents pour le
MFT ont été implémentés.

Dans Waarp Gateway, WebDAV se décline en 2 protocoles distincts :

- *WebDAV over HTTP*, appelé "webdav", opérant par dessus une connexion HTTP claire
- *WebDAV over HTTPS*, appelé "webdav-tls" opérant par dessus une connexion HTTPS chiffrée

Authentification
----------------

L'authentification se fait via le mécanisme d'authentification basique de HTTP
où les identifiants sont transmis encodés en base64 via l'entête "Authorization"
(voir :rfc:`7617` pour plus de détails).

Fonctionnalités
---------------

Côté serveur, Waarp Gateway supporte toutes les opérations permises par WebDAV
excepté :

- la suppression (méthode *DELETE*)
- la copie (méthodes *COPY* et *MOVE*)
- la modification de propriétés (méthode *PROPPATCH*)

Ces fonctionnalités sont disponibles, à la place, via les traitements de Waarp Gateway.

Toutes les autres fonctionnalités de WebDAV (dépot/récupération de fichier,
création de dossier, listage de fichiers, ...) sont, elles, bien implémentées.

Côté client, Waarp Gateway permet le dépot/récupération de fichier, la création
de dossier, et la suppression de fichiers/dossiers.