Remplacer un compte distant
===========================

.. http:put:: /api/partners/(string:partner_name)/accounts/(string:login)

   Remplace le compte donné par celui renseigné en JSON.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson string login: Le login du compte
   :reqjson string password: Le mot de passe du compte

   :statuscode 201: Le compte a été remplacé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte demandé n'existe pas

   :resheader Location: Le chemin d'accès au compte modifié


   **Exemple de requête**

   .. code-block:: http

      PUT https://my_waarp_gateway.net/api/partners/waarp_sftp/accounts/titi HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
      Content-Type: application/json
      Content-Length: 96

      {
        "login": "titi_new",
        "password": "titi_new"
      }

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 201 CREATED
      Location: https://my_waarp_gateway.net/api/partners/waarp_sftp/accounts/titi_new
