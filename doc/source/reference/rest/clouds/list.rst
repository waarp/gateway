Lister les instances cloud
==========================

.. http:get:: /api/clouds

   Renvoie une liste des instances clouds connues.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les utilisateurs seront triés *(défaut: name+)*
   :type sort: [name+|name-|type+|type-]

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Requête invalide
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action

   :resjson array clouds: La liste des instances cloud demandées
   :resjsonarr string name: Le nom de l'instance cloud.
   :resjsonarr string type: Le type d'instance cloud. Voir la section
      :ref:`cloud <reference-cloud>` pour la liste des types d'instance cloud
      supportés.:resjson string key: La clé d'authentification de l'instance cloud (si
      l'instance cloud requiert une authentification).

   :resjsonarr object options: Les options de connexion à l'instance cloud. Voir
      la section :ref:`cloud <reference-cloud>` pour avoir la liste des options
      disponibles pour le type concerné.

   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/clouds?limit=10&sort=name- HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 397

         {
           "clouds": [{
             "name": "ms-azure",
             "type": "azure",
             "key": "bar",
             "options": {
                 "region": "us-east-1"
             }
           },{
             "name": "aws",
             "type": "s3",
             "key": "foo",
             "options": {
                 "region": "eu-west-1"
             }
           }]
         }