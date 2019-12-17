Lister les utilisateurs
=======================

.. http:get:: /api/users

   Renvoie une liste des utilisateurs remplissant les critères donnés en
   paramètres de requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les utilisateurs seront triés *(défaut: username+)*
   :type sort: [username+|username-]
   :param order: L'ordre dans lequel les utilisateurs sont triés *(défaut: asc)*
   :type order: [asc|desc]

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/users?limit=10&order=desc HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array users: La liste des utilisateur demandés
   :resjsonarr number id: L'identifiant unique de l'utilisateur
   :resjsonarr string username: Le nom de l'utilisateur

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 116

          {
            "users": [{
              "id": 2,
              "username": "tutu",
            },{
              "id": 1,
              "username": "toto",
            }]
          }