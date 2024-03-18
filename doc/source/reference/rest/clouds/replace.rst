Remplacer une instance cloud
============================

.. http:put:: /api/clouds/(string:name)

   Remplace l'instance cloud demandée par une nouvelle. Revient à supprimer
   l'ancienne instance, puis d'en insérer une nouvelle. Les champs omis
   reprendront leur valeur par défaut (mise à jour complète).

   :reqheader Authorization: Les identifiants de l'utilisateur REST

   :reqjson string name: Le nouveau nom de l'instance cloud.
   :reqjson string type: Le nouveau type de l'instance cloud. Voir la section
      :ref:`cloud <reference-cloud>` pour la liste des types d'instance cloud
      supportés.
   :reqjson string key: La nouvelle clé d'authentification de l'instance cloud
      (si l'instance cloud requiert une authentification).
   :reqjson string secret: Le nouveau secret d'authentification (mot de passe,
      token...) de l'instance cloud (si l'instance cloud requiert une
      authentification).
   :reqjson object options: Les nouvelles options de connexion à l'instance
      cloud. Voir la section :ref:`cloud <reference-cloud>` pour avoir la liste
      des options disponibles pour le type concerné. **Attention**: la totalité
      des options doit être renseignée. Les options omises seront supprimées.

   :statuscode 201: L'instance cloud a été modifiée avec succès
   :statuscode 400: Requête invalide
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action
   :statuscode 404: L'instance cloud demandée n'existe pas

   :resheader Location: Le chemin d'accès à l'instance cloud mise à jour


   |

   **Exemple de requête**

      .. code-block:: http

         PATCH https://my_waarp_gateway.net/api/cloud/aws HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 114

         {
           "name": "aws-us",
           "type": "s3",
           "key": "bar",
           "options": {
             "region": "us-east-1",
           }
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/clouds/aws-us