Consulter un compte local
=========================

.. http:get:: /api/local_accounts/(int:account_id)

   Renvoie le compte local portant le numéro ``account_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Example:
       .. code-block:: http

          GET https://my_waarp_gateway.net/api/local_accounts/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: Le compte a été renvoyé avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte demandé n'existe pas

   :resjson number id: L'identifiant unique du compte
   :resjson number localAgentID: L'identifiant unique du serveur local auquel
      le compte est rattaché
   :resjson string login: Le login du compte

   :Example:
       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 59

          {
            "id": 1,
            "localAgentID": 1,
            "login": "toto"
          }