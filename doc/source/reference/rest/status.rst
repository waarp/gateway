Statut du service
=================

.. http:get:: /api/status

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 200: Le service est actif
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson object Admin: Le service d'administration de Gateway

      * ``state`` (*string*) - L'état du service
      * ``reason`` (*string*) - En cas d'erreur, donne la cause de l'erreur

   :resjson object Database: Le service de base de données

      * ``state`` (*string*) - L'état du service
      * ``reason`` (*string*) - En cas d'erreur, donne la cause de l'erreur

   :resjson object Controller: Le contrôleur des transferts sortants

      * ``state`` (*string*) - L'état du service
      * ``reason`` (*string*) - En cas d'erreur, donne la cause de l'erreur

   :resjson object {serveur}: Un des serveur de Gateway. Un nouveau champ est
      ajouté pour chaque serveur.

      * ``state`` (*string*) - L'état du service
      * ``reason`` (*string*) - En cas d'erreur, donne la cause de l'erreur

   **Exemple de requête**

   .. code-block:: http

      GET https://my_waarp_gateway.net/api/status HTTP/1.1
      Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

   .. code-block:: http

      HTTP/1.1 200 OK
      Content-Type: application/json
      Content-Length: 212

      {
        "Admin": {
          "state": "Running",
          "reason": ""
        },
        "Database": {
          "state": "Error",
          "reason": "Exemple de message d'erreur"
        },
        "Controller": {
          "state": "Offline",
          "reason": ""
        },
        "serveur_sftp": {
          "state": "Running",
          "reason": ""
        }
      }
