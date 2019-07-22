Statut du service
#################

Afficher le statut du service
=============================

.. http:get:: /api/status

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :Example:
       .. code-block:: http

          GET /api/status HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==



   **Réponse**

   :statuscode 200: Le service est actif
   :statuscode 401: Authentification d'utilisateur invalide

   :Response JSON Object:
       * **Admin** (*object*) - Le statut du service d'administration

           * **State** (*string*) - L'état du service
           * **Reason** (*string*) - En cas d'erreur, donne la cause de l'erreur

       * **Database** (*object*) - Le statut du service de base de données

           * **State** (*string*) - L'état du service
           * **Reason** (*string*) - En cas d'erreur, donne la cause de l'erreur

   :Example:
       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 41

          {
            "Admin": {
              "State": "Running",
              "Reason": ""
            }
          }