.. _ref-proto-http:

============
HTTP & HTTPS
============

L'implémentation de HTTP et HTTPS dans la *Gateway* ajoute quelques fonctionnalités
supplémentaire par rapport au standard. À l'exception d'une seule, toutes ces
fonctionnalités non-standard sont optionnelles, afin de permettre une compatibilité
maximale. Notez également que, bien que semblables, HTTP et HTTPS sont considérés
comme des protocoles séparés par la *gateway*.

**Authentification**

L'authentification du client peut se faire de 2 manières:

- authentification HTTP basique (via un mot de passe)
- via un certificat (uniquement disponible en HTTPS)

Il est à noter que même lorsqu'il s'authentifie via un certificat, le client
**doit** fournit son login via le header d'authentification basique. Si un
certificat est fourni, le mot de passe n'est pas obligatoire, mais il sera
vérifié si un est donné.

.. warning:: TLS version 1.2 est requis au minimum pour utiliser HTTPS. Toutes
   les versions antérieures seront refusées lors de la négociation.

**Requête**

Pour initier un transfert, le client HTTP doit envoyer une requête au serveur.
La méthode de la requête définit le sens du transfert (*POST* pour le sens
client->serveur, et *GET* pour le sens client<-serveur).

Le client doit **impérativement** fournir le nom de la règle à utiliser pour le
transfert, sinon la requête sera refusée. Il dispose pour cela de 2 moyens :
via l'entête ``Waarp-Rule-Name``, ou bien via le paramètre d'URL ``rule``.
Optionnellement, le client peut également fournir l'identifiant du transfert,
qui sera utile en cas d'interruption pour reprendre le transfert. Comme le nom de
règle, cet ID peut également être transmis via un entête (``Waarp-Transfer-ID``)
ou bien via un paramètre d'URL (``id``).

En plus de ces fonctionnalités, le serveur et le client de la *gateway* utilisent
également des fonctionnalités standard du protocole pour transmettre des informations
sur le transfert. Ils font notamment usage des entêtes ``Range`` et ``Content-Range``
pour spécifier la taille du fichier transféré, ainsi que l'endroit à partir duquel
un transfert doit être repris lors d'une reprise de transfert.

Une autre fonctionnalité utilisée par le client et le serveur est les requêtes
*HEAD*. Il est en effet possible pour un client de demander des informations sur
le statut d'un transfert en envoyant une requête *HEAD* accompagnée de l'identifiant
du transfert. Cela est notamment utilisé par le client de la *gateway* dans certains
cas pour déterminer où un transfert doit être repris (à noter que si le serveur ne
supporte pas la fonctionnalité, le client reprendra le transfert du début).

**Fin de transfert**

Lorsque le transfert se termine (que ce soit normalement ou bien à cause d'une
erreur), le serveur renvoie les entêtes suivant dans sa réponse afin de communiquer
l'état final du transfert :

- ``Waarp-Transfer-Status`` renseignant le statut final du transfert
- ``Waarp-Error-Code`` donnant le code d'erreur du transfer (si une erreur s'est
  produite)
- ``Waarp-Error-Message`` donnant le message d'erreur du transfer (si une erreur
  s'est produite)

À noter que si le serveur est envoyeur du fichier (requête *GET*), alors ces
entêtes seront envoyés en fin de message (via les *trailers* HTTP) après le corps
contenant le fichier. De même, si le client est l'envoyeur du fichier (requête
*POST*), le statut final du transfert sera envoyé à la fin de la requête via les
*trailers*.