.. _ref-proto-http:

============
HTTP & HTTPS
============

L'implémentation de HTTP et HTTPS dans Waarp Gateway ajoute quelques fonctionnalités
supplémentaire par rapport au standard. À l'exception d'une seule, toutes ces
fonctionnalités non-standard sont optionnelles, afin de permettre une compatibilité
maximale. Notez également que, bien que semblables, HTTP et HTTPS sont considérés
comme des protocoles séparés par Waarp Gateway.

Authentification
----------------

L'authentification du client peut se faire de 2 manières:

- authentification HTTP basique (via un mot de passe)
- via un certificat (uniquement disponible en HTTPS)

Il est à noter que même lorsqu'il s'authentifie via un certificat, le client
**doit** fournit son login via le header d'authentification basique. Si un
certificat est fourni, le mot de passe n'est pas obligatoire, mais il sera
vérifié si un est donné.

.. warning::
   TLS version 1.2 est requis au minimum pour utiliser HTTPS. Toutes
   les versions antérieures seront refusées lors de la négociation.

Requête
-------

Pour initier un transfert, le client HTTP doit envoyer une requête au serveur.
La méthode de la requête définit le sens du transfert (:http:method:`POST` pour le sens
client->serveur, et :http:method:`GET` pour le sens client<-serveur).

Le client doit **impérativement** fournir le nom de la règle à utiliser pour le
transfert, sinon la requête sera refusée. Il dispose pour cela de 2 moyens : via
l'entête :http:header:`Waarp-Rule-Name`, ou bien via le paramètre d'URL
``rule``. Optionnellement, le client peut également fournir l'identifiant du
transfert, qui sera utile en cas d'interruption pour reprendre le transfert.
Comme le nom de règle, cet ID peut également être transmis via un entête
(:http:header:`Waarp-Transfer-ID`) ou bien via un paramètre d'URL (``id``).

En plus de ces fonctionnalités, le serveur et le client de Waarp Gateway
utilisent également des fonctionnalités standard du protocole pour transmettre
des informations sur le transfert. Ils font notamment usage des entêtes
:http:header:`Range` et :http:header:`Content-Range` pour spécifier la taille
du fichier transféré, ainsi que l'endroit à partir duquel un transfert doit être
repris lors d'une reprise de transfert.

Une autre fonctionnalité utilisée par le client et le serveur est les requêtes
:http:method:`HEAD`:. Il est en effet possible pour un client de demander des
informations sur le statut d'un transfert en envoyant une requête
:http:method:`HEAD`: accompagnée de l'identifiant du transfert. Cela est
notamment utilisé par le client de Waarp Gateway dans certains cas pour
déterminer où un transfert doit être repris (à noter que si le serveur ne
supporte pas la fonctionnalité, le client reprendra le transfert du début).

HTTP supporte également l'envoi d':term:`infos de transfert` via l'entête
spécial :http:header:`Waarp-Transfer-Info`. Chaque paire doit être présentée
sous la forme d'une liste de ``<clé>=<valeur>`` avec ``<clé>`` le nom de la clé,
et ``<valeur>`` la valeur encodée en JSON. Les paires doivent soit être séparées
par une virgule. Alternativement, l'entête peut également être répété pour
chaque paire.

Fin de transfert
----------------

Lorsque le transfert se termine (que ce soit normalement ou bien à cause d'une
erreur), le serveur renvoie les entêtes suivant dans sa réponse afin de communiquer
l'état final du transfert :

- :http:header:`Waarp-Transfer-Status` renseignant le statut final du transfert
- :http:header:`Waarp-Error-Code` donnant le code d'erreur du transfert (si une
  erreur s'est produite)
- :http:header:`Waarp-Error-Message` donnant le message d'erreur du transfert
  (si une erreur s'est produite)

À noter que si Gateway agit en tant que serveur et est envoyeur du fichier
(requête :http:method:`GET`), alors ces entêtes seront envoyés en fin de message
(via les *trailers* HTTP) après le corps contenant le fichier. De même, si
Gateway agit comme client et  est l'envoyeur du fichier (requête
:http:method:`POST`), le statut final du transfert sera envoyé à la fin de la
requête via les *trailers*.
