Arrêter un filewatcher
======================

.. http:put:: /api/filewatchers/(string:flow)/stop

   Arrête le *filewatcher* demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :statuscode 202: Le *filewatcher* a été arrêté avec succès
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action
   :statuscode 404: Le *filewatcher* demandé n'existe pas

   |

   **Exemple de requête**

      .. code-block:: http

         PUT https://my_waarp_gateway.net/api/filewatchers/my-filewatcher/stop HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 202 ACCEPTED
