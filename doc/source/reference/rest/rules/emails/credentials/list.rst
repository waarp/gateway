Lister les identifiants SMTP
============================

.. http:get:: /api/email/credentials

   Renvoie la liste des identifiants SMTP.

   :reqheader Authorization: Les identifiants de l'utilisateur.

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sortby: Le paramètre selon lequel les règles seront triées *(défaut: name+)*
   :type sortby: [name+|name-]

   :statuscode 200: L'identifiant a été renvoyé avec succès.
   :statuscode 401: Authentification d'utilisateur invalide.
   :statuscode 404: L'identifiant demandé n'existe pas.

   :resjsonarr string emailAddress: L'adresse email d'envoi.
   :resjsonarr string serverAddress: L'adresse (port inclus) du serveur SMTP servant
     à envoyer les emails.
   :resjsonarr string login: Le nom d'utilisateur à utiliser pour l'authentification SMTP.
   :resjsonarr string password: Le mot de passe utilisateur.


   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/email/credentials HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 436

      [{
        "emailAddress": "waarp@example.com",
        "serverAddress": "smtp.example.com:587",
        "login": "waarp",
        "password": "sesame"
      }, {
        "emailAddress": "prod@example.com",
        "serverAddress": "smtp.example.com:587",
        "login": "waarp",
        "password": "sesame"
      }]
