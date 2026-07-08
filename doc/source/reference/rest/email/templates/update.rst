Modifier un template d'email
============================

.. http:patch:: /api/email/templates/(string:name)

   Met à jour le template d'email demandé. Les champs omis resteront inchangés
   (mise à jour partielle).

   :reqheader Authorization: Les identifiants de l'utilisateur REST

   :reqjson string name: Le nouveau nom du template.
   :reqjson string subject: Le nouveau sujet de l'email.
   :reqjson string mimeType: Le nouveau type MIME du corps de l'email.
   :reqjson string body: Le nouveau corps de l'email.
   :reqjson array attachments: La nouvelle liste des fichiers joints au template.

   :statuscode 201: Le template a été mis à jour avec succès
   :statuscode 400: Requête invalide
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action
   :statuscode 404: Le template demandé n'existe pas

   :resheader Location: Le chemin d'accès au template mis à jour

   |

   **Exemple de requête**

      .. code-block:: http

         PATCH https://my_waarp_gateway.net/api/email/templates/transfer-error HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 52

         {
           "subject": "Échec de transfert Gateway"
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/email/templates/transfer-error
