Modifier un compte local
========================

.. http:put:: /api/local_accounts/(int:account_id)

   Met à jour le compte local portant le numéro ``account_id`` avec les informations
   renseignées en format JSON dans le corps de la requête. Les champs non-spécifiés
   resteront inchangés.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :reqjson number localAgentID: L'identifiant unique du serveur local auquel
      le compte est rattaché
   :reqjson string login: Le login du compte
   :reqjson string password: Le mot de passe du compte

   **Exemple de requête**

       .. code-block:: http

          PATCH https://my_waarp_gateway.net/api/local_accounts/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
          Content-Type: application/json
          Content-Length: 96

          {
            "localAgentID": 2,
            "login": "toto_new",
            "password": "titi_new"
          }


   **Réponse**

   :statuscode 201: Le compte a été remplacé avec succès
   :statuscode 400: Un ou plusieurs des paramètres du compte sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le compte demandé n'existe pas

   :resheader Location: Le chemin d'accès au compte modifié

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 201 CREATED
          Location: https://my_waarp_gateway.net/api/local_accounts/1